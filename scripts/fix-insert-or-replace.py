#!/usr/bin/env python3
"""
Convert SQLite INSERT OR REPLACE to PostgreSQL INSERT ... ON CONFLICT
"""

import re
from pathlib import Path

def fix_insert_or_replace(filepath):
    """Fix INSERT OR REPLACE in a single file"""
    content = filepath.read_text(encoding='utf-8')
    original = content
    
    if 'INSERT OR REPLACE' not in content:
        return False
    
    # Replace INSERT OR REPLACE with INSERT
    content = content.replace('INSERT OR REPLACE INTO changelogs', 'INSERT INTO changelogs')
    
    # Find all INSERT statements and add ON CONFLICT
    # Pattern: INSERT INTO changelogs (...) VALUES (...);
    pattern = r"(INSERT INTO changelogs\s*\([^)]+\)\s*VALUES\s*\([^;]+\));"
    
    def add_on_conflict(match):
        insert_stmt = match.group(1)
        return f"{insert_stmt}\nON CONFLICT (version) DO UPDATE SET\n    release_date = EXCLUDED.release_date,\n    content = EXCLUDED.content;"
    
    content = re.sub(pattern, add_on_conflict, content, flags=re.DOTALL | re.MULTILINE)
    
    if content != original:
        filepath.write_text(content, encoding='utf-8')
        return True
    
    return False

def main():
    migrations_dir = Path("E:/golang/my_projects/internal/storage/postgresql/migrations")
    
    print("Converting INSERT OR REPLACE to PostgreSQL syntax...\n")
    
    fixed = 0
    for sql_file in sorted(migrations_dir.glob("*.up.sql")):
        if fix_insert_or_replace(sql_file):
            print(f"[OK] {sql_file.name}")
            fixed += 1
    
    print(f"\nTotal files fixed: {fixed}")

if __name__ == '__main__':
    main()

