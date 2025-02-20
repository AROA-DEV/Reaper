import os
import time
import json
import hashlib
import magic
import pandas as pd
import plotly.express as px
from watchdog.observers import Observer
from watchdog.events import FileSystemEventHandler
from collections import defaultdict
import shutil
import logging
from datetime import datetime
import platform
import subprocess
import string
from Harvester.target_meta import create_target_metadata, save_target_metadata
from rich.console import Console
from rich.table import Table
from rich.progress import Progress, SpinnerColumn, TextColumn
from rich.live import Live
from rich import print as rprint
from http.server import HTTPServer, SimpleHTTPRequestHandler
import threading
import webbrowser
import jinja2
from flask import Flask, render_template_string

class USBMonitor(FileSystemEventHandler):
    def __init__(self):
        self.data_store = defaultdict(list)
        self.setup_logging()
        self.target_dir = os.path.expanduser("~/Documents/Harvested_Data")
        os.makedirs(self.target_dir, exist_ok=True)
        self.platform = platform.system().lower()
        self.known_drives = set(self.get_drive_list())
        self.console = Console()
        self.web_server = None
        self.server_port = 8000
        self.flask_app = Flask(__name__)
        self.setup_routes()
        self.server_thread = None
        
    def setup_logging(self):
        logging.basicConfig(
            filename='harvester.log',
            level=logging.INFO,
            format='%(asctime)s - %(levelname)s - %(message)s'
        )

    def copy_and_remove(self, src_path, file_info, target_dir):
        """Copy file to target directory and remove from source"""
        try:
            # Create date-based subdirectory within target directory
            date_dir = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
            dest_dir = os.path.join(target_dir, date_dir)
            os.makedirs(dest_dir, exist_ok=True)
            
            # Copy file with hash as part of filename
            filename = os.path.basename(src_path)
            base, ext = os.path.splitext(filename)
            dest_path = os.path.join(dest_dir, f"{base}_{file_info['hash'][:8]}{ext}")
            
            shutil.copy2(src_path, dest_path)
            os.remove(src_path)  # Remove original file
            logging.info(f"Copied and removed: {src_path} -> {dest_path}")
            return dest_path
        except Exception as e:
            logging.error(f"Error copying/removing file {src_path}: {str(e)}")
            return None

    def setup_routes(self):
        @self.flask_app.route('/')
        def index():
            return render_template_string("""
                <html>
                <head>
                    <title>USB Harvester Dashboard</title>
                    <style>
                        body { font-family: Arial, sans-serif; margin: 20px; }
                        .nav { margin-bottom: 20px; }
                        .nav a { margin-right: 10px; }
                    </style>
                </head>
                <body>
                    <h1>USB Harvester Dashboard</h1>
                    <div class="nav">
                        <a href="/types">File Types</a>
                        <a href="/sizes">File Sizes</a>
                        <a href="/importance">Importance Scores</a>
                    </div>
                    <div id="content">
                        Select a view from the navigation above.
                    </div>
                </body>
                </html>
            """)

    def start_web_server(self, directory):
        """Start Flask web server"""
        if self.server_thread and self.server_thread.is_alive():
            return
        
        def run_server():
            self.flask_app.run(host='localhost', port=self.server_port)
            
        self.server_thread = threading.Thread(target=run_server)
        self.server_thread.daemon = True
        self.server_thread.start()
        
        dashboard_url = f"http://localhost:{self.server_port}"
        self.console.print(f"\n[green]Dashboard available at:[/green] {dashboard_url}")
        webbrowser.open(dashboard_url)

    def scan_files(self, path):
        """Scan files and collect metadata"""
        total_size = 0
        file_data = []
        
        with Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            transient=True
        ) as progress:
            # Calculate total size
            size_task = progress.add_task("Calculating total size...", total=None)
            for root, _, files in os.walk(path):
                for file in files:
                    full_path = os.path.join(root, file)
                    try:
                        stats = os.stat(full_path)
                        total_size += stats.st_size
                    except Exception:
                        continue
            progress.remove_task(size_task)

            # Create target metadata
            target_meta = create_target_metadata(total_size)
            
            # Create target-specific directory
            target_dir = os.path.join(
                self.target_dir, 
                f"{target_meta['system_info']['hostname']}_{target_meta['system_info']['username']}"
            )
            os.makedirs(target_dir, exist_ok=True)
            
            # Save target metadata
            save_target_metadata(target_meta, target_dir)

            # Process files
            file_task = progress.add_task("Processing files...")
            for root, _, files in os.walk(path):
                for file in files:
                    full_path = os.path.join(root, file)
                    try:
                        stats = os.stat(full_path)
                        file_info = {
                            'path': full_path,
                            'size': stats.st_size,
                            'created': datetime.fromtimestamp(stats.st_ctime),
                            'modified': datetime.fromtimestamp(stats.st_mtime),
                            'file_type': magic.from_file(full_path),
                            'extension': os.path.splitext(file)[1],
                            'hash': self.get_file_hash(full_path)
                        }
                        file_info['importance_score'] = self.calculate_importance(file_info)
                        
                        # Update destination path to use target directory
                        new_path = self.copy_and_remove(full_path, file_info, target_dir)
                        if new_path:
                            file_info['original_path'] = file_info['path']
                            file_info['path'] = new_path
                            file_data.append(file_info)
                    except Exception as e:
                        logging.error(f"Error processing file {full_path}: {str(e)}")
            progress.remove_task(file_task)

        return file_data, target_meta

    def get_file_hash(self, filepath):
        """Calculate SHA-256 hash of file"""
        hasher = hashlib.sha256()
        with open(filepath, 'rb') as f:
            for chunk in iter(lambda: f.read(4096), b''):
                hasher.update(chunk)
        return hasher.hexdigest()

    def calculate_importance(self, file_info):
        """Calculate importance score based on multiple factors"""
        score = 0
        
        # Size factor
        if file_info['size'] > 1024*1024: # >1MB
            score += 1
            
        # Extension importance
        important_extensions = ['.doc', '.docx', '.pdf', '.xls', '.xlsx', '.txt']
        if file_info['extension'].lower() in important_extensions:
            score += 2
            
        # Recent modification
        if (datetime.now() - file_info['modified']).days < 30:
            score += 1
            
        return score

    def generate_dashboard(self, data, target_meta):
        """Generate dashboard using plotly"""
        with self.console.status("[bold green]Generating dashboard..."):
            df = pd.DataFrame(data)
            dashboard_dir = os.path.join(
                self.target_dir,
                f"{target_meta['system_info']['hostname']}_{target_meta['system_info']['username']}",
                "dashboard"
            )
            os.makedirs(dashboard_dir, exist_ok=True)

            # Update Flask routes for the visualizations
            @self.flask_app.route('/types')
            def file_types():
                fig = px.pie(df, names='file_type', title='File Type Distribution')
                return fig.to_html(full_html=True)

            @self.flask_app.route('/sizes')
            def file_sizes():
                fig = px.histogram(df, x='size', title='File Size Distribution')
                return fig.to_html(full_html=True)

            @self.flask_app.route('/importance')
            def importance_scores():
                fig = px.bar(df, x='path', y='importance_score', title='File Importance Scores')
                return fig.to_html(full_html=True)

            # Save summary data
            summary = {
                'total_files': len(data),
                'total_size': sum(f['size'] for f in data),
                'avg_importance': sum(f['importance_score'] for f in data) / len(data),
                'scan_time': datetime.now().isoformat(),
                "target_info": target_meta
            }
            
            with open(os.path.join(dashboard_dir, 'scan_summary.json'), 'w') as f:
                json.dump(summary, f, indent=4)

            # Start web server and display statistics
            self.start_web_server(dashboard_dir)
            self.display_statistics(data)

    def display_statistics(self, data):
        """Display file statistics in terminal"""
        table = Table(show_header=True, header_style="bold magenta")
        table.add_column("Statistic", style="dim")
        table.add_column("Value")
        
        table.add_row("Total Files", str(len(data)))
        table.add_row("Total Size", f"{sum(f['size'] for f in data) / (1024*1024):.2f} MB")
        table.add_row("Average Importance", 
                     f"{sum(f['importance_score'] for f in data) / len(data):.2f}")
        
        self.console.print("\n[bold]File Statistics:[/bold]")
        self.console.print(table)

    def get_drive_list(self):
        """Get list of current drives based on platform"""
        if self.platform == 'windows':
            return self.get_windows_drives()
        elif self.platform == 'linux':
            return self.get_linux_drives()
        elif self.platform == 'darwin':
            return self.get_macos_drives()
        return []

    def get_windows_drives(self):
        """Get Windows removable drives"""
        try:
            import win32file
            drives = []
            for letter in string.ascii_uppercase:
                drive = f"{letter}:"
                try:
                    drive_type = win32file.GetDriveType(drive)
                    if (drive_type == win32file.DRIVE_REMOVABLE or 
                        drive_type == win32file.DRIVE_FIXED):
                        drives.append(drive)
                except:
                    continue
            return drives
        except ImportError:
            logging.error("win32file module not available")
            return []

    def get_linux_drives(self):
        """Get Linux USB drives"""
        try:
            result = subprocess.run(['lsblk', '-Jpo', 'NAME,TRAN'], 
                                 capture_output=True, text=True)
            if result.returncode == 0:
                import json
                data = json.loads(result.stdout)
                return [dev['name'] for dev in data['blockdevices'] 
                       if dev.get('tran') == 'usb']
            return []
        except Exception as e:
            logging.error(f"Error detecting Linux drives: {e}")
            return []

    def get_macos_drives(self):
        """Get macOS USB drives"""
        try:
            result = subprocess.run(['diskutil', 'list', '-plist', 'external'], 
                                 capture_output=True, text=True)
            if result.returncode == 0:
                import plistlib
                data = plistlib.loads(result.stdout.encode())
                return [disk for disk in data.get('AllDisksAndPartitions', [])]
            return []
        except Exception as e:
            logging.error(f"Error detecting macOS drives: {e}")
            return []

    def normalize_path(self, path):
        """Normalize path based on platform"""
        return os.path.normpath(path)

    def check_for_new_drives(self):
        """Check for newly connected drives"""
        with Live(self.console.status("[bold blue]Monitoring for USB drives...[/bold blue]"),
                 refresh_per_second=4) as live:
            while True:
                current_drives = set(self.get_drive_list())
                new_drives = current_drives - self.known_drives
                
                for drive in new_drives:
                    self.console.print(f"[green]New USB drive detected:[/green] {drive}")
                    try:
                        mount_point = self.get_mount_point(drive)
                        if mount_point:
                            data, target_meta = self.scan_files(mount_point)
                            self.data_store[drive] = data
                            self.generate_dashboard(data, target_meta)
                            logging.info(f"Completed scan of {drive}")
                    except Exception as e:
                        logging.error(f"Error processing drive {drive}: {str(e)}")
                
                self.known_drives = current_drives
                time.sleep(2)

    def get_mount_point(self, drive):
        """Get mount point based on platform"""
        if self.platform == 'windows':
            return f"{drive}\\"
        elif self.platform == 'linux':
            try:
                result = subprocess.run(['findmnt', '-n', '-o', 'TARGET', drive],
                                     capture_output=True, text=True)
                return result.stdout.strip()
            except:
                return None
        elif self.platform == 'darwin':
            try:
                result = subprocess.run(['diskutil', 'info', '-plist', drive],
                                     capture_output=True, text=True)
                if result.returncode == 0:
                    import plistlib
                    data = plistlib.loads(result.stdout.encode())
                    return data.get('MountPoint')
            except:
                return None
        return None

    def on_created(self, event):
        """This is no longer used as we're using direct drive detection"""
        pass

def main():
    console = Console()
    
    with console.screen() as screen:
        console.print("[bold]USB Harvester[/bold]", justify="center")
        console.print("[dim]Press Ctrl+C to exit[/dim]\n", justify="center")
        
        try:
            monitor = USBMonitor()
            monitor.check_for_new_drives()
        except KeyboardInterrupt:
            console.print("\n[yellow]Shutting down...[/yellow]")
        finally:
            if monitor.web_server:
                monitor.web_server.shutdown()

if __name__ == "__main__":
    main()
