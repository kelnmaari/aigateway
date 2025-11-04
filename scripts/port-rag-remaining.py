#!/usr/bin/env python3
"""
Скрипт для портирования оставшихся RAG методов из SQLite в PostgreSQL.
Заменяет ? на $N плейсхолдеры и boolToInt на boolean.
"""

import re
import sys

def convert_placeholders(sql_code):
    """Заменяет ? плейсхолдеры на $N для PostgreSQL"""
    result = []
    param_counter = 1
    
    for line in sql_code.split('\n'):
        # Если в строке есть ?, заменяем последовательно на $1, $2, $3...
        while '?' in line:
            line = line.replace('?', f'${param_counter}', 1)
            param_counter += 1
        result.append(line)
    
    return '\n'.join(result)

def convert_bool_calls(code):
    """Убирает boolToInt() и intToBool() вызовы"""
    # boolToInt(source.IsShared) -> source.IsShared
    code = re.sub(r'boolToInt\(([^)]+)\)', r'\1', code)
    # source.IsShared = intToBool(isSharedInt) -> source.IsShared = isSharedBool
    code = re.sub(r'(\w+)\s*=\s*intToBool\((\w+)\)', r'\1 = \2', code)
    # var isSharedInt int -> var isSharedBool bool
    code = re.sub(r'var (\w+)Int int', r'var \1Bool bool', code)
    return code

def convert_json_strings(code):
    """Заменяет string(jsonData) на jsonData для JSONB"""
    code = re.sub(r'string\((\w+JSON)\)', r'\1', code)
    return code

def main():
    print("Скрипт готов для портирования RAG методов")
    print("Используйте этот шаблон для конвертации:")
    print("1. Замена ? -> $N")
    print("2. Убрать boolToInt/intToBool")
    print("3. JSONB колонки: []byte вместо string")
    print("4. Замена s.db -> db.db и s.logger -> db.logger")

if __name__ == "__main__":
    main()

