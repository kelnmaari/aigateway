#!/usr/bin/env python3
"""Fix remaining PostgreSQL conversion errors"""

import re
from pathlib import Path

def fix_file(filepath):
    """Fix all conversion errors in a file"""
    content = filepath.read_text(encoding='utf-8')
    original = content
    
    # 1. Replace standalone s. at beginning of line or after whitespace to db.
    content = re.sub(r'(\W)s\.(db|logger|scan)', r'\1db.\2', content)
    
    # 2. Replace TRUE/FALSE with true/false
    content = re.sub(r'\bTRUE\b', 'true', content)
    content = re.sub(r'\bFALSE\b', 'false', content)
    
    # 3. Fix rowsAffected == false to rowsAffected == 0
    content = re.sub(r'rowsAffected == false', 'rowsAffected == 0', content)
    content = re.sub(r'rowsAffected == true', 'rowsAffected != 0', content)
    
    # Only write if changed
    if content != original:
        filepath.write_text(content, encoding='utf-8')
        return True
    return False

def main():
    postgres_dir = Path("E:/golang/my_projects/internal/storage/postgresql")
    
    fixed = 0
    for gofile in postgres_dir.glob("*.go"):
        if fix_file(gofile):
            print(f"Fixed: {gofile.name}")
            fixed += 1
    
    print(f"\nFixed {fixed} files.")

if __name__ == '__main__':
    main()

