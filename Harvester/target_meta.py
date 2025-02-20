import json
import os
import platform
import getpass
from datetime import datetime

def create_target_metadata(export_size):
    """Create metadata for the current target system"""
    metadata = {
        "system_info": {
            "hostname": platform.node(),
            "system": platform.system(),
            "release": platform.release(),
            "version": platform.version(),
            "machine": platform.machine(),
            "processor": platform.processor(),
            "username": getpass.getuser(),
        },
        "export_info": {
            "total_size": export_size,
            "export_date": datetime.now().isoformat(),
            "export_id": f"{platform.node()}_{getpass.getuser()}_{datetime.now().strftime('%Y%m%d_%H%M%S')}"
        }
    }
    return metadata

def save_target_metadata(metadata, destination_path):
    """Save target metadata to specified path"""
    meta_file = os.path.join(destination_path, "target_meta.json")
    with open(meta_file, 'w') as f:
        json.dump(metadata, f, indent=4)
    return meta_file
