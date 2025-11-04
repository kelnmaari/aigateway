#!/usr/bin/env python3
"""
Convert SQLite CRUD files to PostgreSQL format
Replaces ? placeholders with $1, $2, $3... and updates syntax
"""

import re
import sys
from pathlib import Path

def replace_placeholders(content):
    """Replace ? placeholders with $1, $2, $3... in SQL queries"""
    
    # Find all SQL query strings (between backticks)
    def replace_in_query(match):
        query = match.group(1)
        counter = 1
        result = []
        i = 0
        
        while i < len(query):
            if query[i] == '?':
                result.append(f'${counter}')
                counter += 1
                i += 1
            else:
                result.append(query[i])
                i += 1
        
        return '`' + ''.join(result) + '`'
    
    # Process SQL strings in backticks
    content = re.sub(r'`([^`]*)`', replace_in_query, content, flags=re.DOTALL)
    
    return content

def convert_file(source_path, target_path):
    """Convert a single SQLite CRUD file to PostgreSQL"""
    
    print(f"Converting: {source_path.name}")
    
    # Read source
    content = source_path.read_text(encoding='utf-8')
    
    # 1. Replace package name
    content = content.replace('package sqlite', 'package postgresql')
    
    # 2. Replace struct receiver
    content = re.sub(r'\(s \*SQLiteDB\)', '(db *PostgreSQLDB)', content)
    content = re.sub(r'\bs\.db\b', 'db.db', content)
    content = re.sub(r'\bs\.logger\b', 'db.logger', content)
    
    # 3. Replace SQLite-specific functions
    content = content.replace('CURRENT_TIMESTAMP', 'NOW()')
    content = re.sub(r'DEFAULT 1([,\s\)])', r'DEFAULT TRUE\1', content)
    content = re.sub(r'DEFAULT 0([,\s\)])', r'DEFAULT FALSE\1', content)
    content = re.sub(r'= 1\b', '= TRUE', content)
    content = re.sub(r'= 0\b', '= FALSE', content)
    
    # 4. Replace placeholders
    content = replace_placeholders(content)
    
    # Write target
    target_path.write_text(content, encoding='utf-8')
    
    print(f"  -> Created: {target_path}")

def main():
    sqlite_dir = Path("E:/golang/my_projects/internal/storage/sqlite")
    postgres_dir = Path("E:/golang/my_projects/internal/storage/postgresql")
    
    # Files to convert
    files_to_convert = [
        "apikeys.go",
        "conversations.go",
        "audit.go",
        "rbac.go",
        "quotas.go",
        "model_registry.go",
        "mcp.go",
        "changelogs.go",
        "files.go",
        "invitations.go",
        "model_configs.go",
        "usage.go"
    ]
    
    converted = 0
    for filename in files_to_convert:
        source = sqlite_dir / filename
        target = postgres_dir / filename
        
        if not source.exists():
            print(f"Skip: {filename} (not found)")
            continue
        
        try:
            convert_file(source, target)
            converted += 1
        except Exception as e:
            print(f"Error converting {filename}: {e}")
    
    print(f"\nDone! Converted {converted}/{len(files_to_convert)} files.")
    print("\nPostgreSQL CRUD files created in:", postgres_dir)

if __name__ == '__main__':
    main()

