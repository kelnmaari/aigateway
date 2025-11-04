#!/usr/bin/env python3
"""
Fix common PostgreSQL migration syntax errors
"""

import re
from pathlib import Path

def fix_migration_file(filepath):
    """Fix syntax errors in a single migration file"""
    content = filepath.read_text(encoding='utf-8')
    original = content
    fixes = []
    
    # Fix 1: INTEGER DEFAULT FALSE/TRUE -> 0/1
    pattern1 = r'(\s+\w+\s+INTEGER[^,;]*DEFAULT\s+)FALSE'
    if re.search(pattern1, content, re.IGNORECASE):
        content = re.sub(pattern1, r'\g<1>0', content, flags=re.IGNORECASE)
        fixes.append("INTEGER DEFAULT FALSE -> 0")
    
    pattern2 = r'(\s+\w+\s+INTEGER[^,;]*DEFAULT\s+)TRUE'
    if re.search(pattern2, content, re.IGNORECASE):
        content = re.sub(pattern2, r'\g<1>1', content, flags=re.IGNORECASE)
        fixes.append("INTEGER DEFAULT TRUE -> 1")
    
    # Fix 2: BOOLEAN DEFAULT 0/1 -> FALSE/TRUE
    pattern3 = r'(\s+\w+\s+BOOLEAN[^,;]*DEFAULT\s+)0\b'
    if re.search(pattern3, content, re.IGNORECASE):
        content = re.sub(pattern3, r'\g<1>FALSE', content, flags=re.IGNORECASE)
        fixes.append("BOOLEAN DEFAULT 0 -> FALSE")
    
    pattern4 = r'(\s+\w+\s+BOOLEAN[^,;]*DEFAULT\s+)1\b'
    if re.search(pattern4, content, re.IGNORECASE):
        content = re.sub(pattern4, r'\g<1>TRUE', content, flags=re.IGNORECASE)
        fixes.append("BOOLEAN DEFAULT 1 -> TRUE")
    
    # Fix 3: JSONB, array -> JSONB, -- array
    pattern5 = r'(JSONB),(\s+)(array[^-])'
    if re.search(pattern5, content, re.IGNORECASE):
        content = re.sub(pattern5, r'\1, --\2\3', content, flags=re.IGNORECASE)
        fixes.append("JSONB, array -> JSONB, -- array")
    
    # Fix 4: Comment without -- after comma (aggressive fix)
    lines = content.split('\n')
    fixed_lines = []
    for line in lines:
        # Check if line has comma followed by text (not SQL keywords)
        if re.search(r',\s+[a-zA-Z_]+\s*\([^)]*\)\s*$', line) and '--' not in line:
            # Add -- before the text in parentheses
            line = re.sub(r',(\s+)([a-zA-Z_]+\s*\([^)]*\))\s*$', r', --\1\2', line)
            if line != lines[fixed_lines.__len__()]:
                fixes.append("Added -- to comment after comma")
        fixed_lines.append(line)
    
    if len(fixed_lines) > 0:
        content = '\n'.join(fixed_lines)
    
    # Only write if changes were made
    if content != original:
        filepath.write_text(content, encoding='utf-8')
        return fixes
    
    return []

def main():
    migrations_dir = Path("E:/golang/my_projects/internal/storage/postgresql/migrations")
    
    print("Fixing PostgreSQL migration syntax errors...\n")
    
    total_fixed = 0
    files_fixed = 0
    
    for sql_file in sorted(migrations_dir.glob("*.up.sql")):
        fixes = fix_migration_file(sql_file)
        
        if fixes:
            files_fixed += 1
            total_fixed += len(fixes)
            print(f"[OK] {sql_file.name}")
            for fix in fixes:
                print(f"   - {fix}")
    
    print(f"\nSummary:")
    print(f"   Files fixed: {files_fixed}")
    print(f"   Total fixes: {total_fixed}")
    
    if files_fixed == 0:
        print("   No errors found!")

if __name__ == '__main__':
    main()

