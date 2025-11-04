#!/usr/bin/env python3
"""
Fix SQLite triggers and DATETIME to PostgreSQL syntax
"""

import re
from pathlib import Path

def fix_datetime(content):
    """Replace DATETIME with TIMESTAMP"""
    return content.replace('DATETIME', 'TIMESTAMP')

def fix_trigger(content):
    """Convert SQLite triggers to PostgreSQL"""
    # Pattern: CREATE TRIGGER IF NOT EXISTS name ... BEGIN ... END;
    pattern = r'CREATE TRIGGER IF NOT EXISTS\s+(\w+)\s+AFTER UPDATE ON\s+(\w+)\s+FOR EACH ROW\s+BEGIN\s+UPDATE\s+\2\s+SET\s+(\w+)\s*=\s*CURRENT_TIMESTAMP\s+WHERE\s+id\s*=\s*NEW\.id;\s+END;'
    
    def replace_trigger(match):
        trigger_name = match.group(1)
        table_name = match.group(2)
        column_name = match.group(3)
        
        return f"""-- Trigger function для автоматического обновления {column_name}
CREATE OR REPLACE FUNCTION {trigger_name}()
RETURNS TRIGGER AS $$
BEGIN
    NEW.{column_name} = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger для автоматического обновления {column_name}
DROP TRIGGER IF EXISTS {trigger_name}_trigger ON {table_name};
CREATE TRIGGER {trigger_name}_trigger
    BEFORE UPDATE ON {table_name}
    FOR EACH ROW
    EXECUTE FUNCTION {trigger_name}();"""
    
    return re.sub(pattern, replace_trigger, content, flags=re.DOTALL | re.MULTILINE)

def fix_file(filepath):
    """Fix a single migration file"""
    content = filepath.read_text(encoding='utf-8')
    original = content
    
    # Apply fixes
    content = fix_datetime(content)
    content = fix_trigger(content)
    
    if content != original:
        filepath.write_text(content, encoding='utf-8')
        return True
    
    return False

def main():
    migrations_dir = Path("E:/golang/my_projects/internal/storage/postgresql/migrations")
    
    # Files that need fixing
    files_to_fix = [
        "062_create_model_registry_tables.up.sql",
        "051_create_rag_tables.up.sql",
        "066_add_device_fields_to_api_keys.up.sql"
    ]
    
    print("Fixing SQLite triggers and DATETIME...\n")
    
    fixed = 0
    for filename in files_to_fix:
        filepath = migrations_dir / filename
        if filepath.exists():
            if fix_file(filepath):
                print(f"[OK] {filename}")
                fixed += 1
        else:
            print(f"[SKIP] {filename} not found")
    
    print(f"\nTotal files fixed: {fixed}")

if __name__ == '__main__':
    main()

