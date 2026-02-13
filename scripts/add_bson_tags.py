#!/usr/bin/env python3
"""为 model 包中的所有 struct 字段添加 bson tag。

规则：
- `json:"id" db:"id"` → `json:"id" bson:"_id" db:"id"`
- `json:"xxx" db:"yyy"` → `json:"xxx" bson:"xxx" db:"yyy"`
- `json:"xxx,omitempty" db:"yyy"` → `json:"xxx,omitempty" bson:"xxx,omitempty" db:"yyy"`
- `json:"-" db:"yyy"` → `json:"-" bson:"yyy" db:"yyy"` (hidden from JSON but stored in DB)
"""
import re
import glob
import os

model_dir = '/home/yun/wp/agents-admin/internal/shared/model'

for filepath in sorted(glob.glob(os.path.join(model_dir, '*.go'))):
    if filepath.endswith('_test.go'):
        continue

    with open(filepath) as f:
        content = f.read()

    if 'bson:"' in content:
        print(f'SKIP (already has bson): {os.path.basename(filepath)}')
        continue

    original = content

    # Pattern: json:"xxx" db:"yyy"
    def replace_json_db(m):
        json_val = m.group(1)
        db_val = m.group(2)

        json_name = json_val.split(',')[0]
        omitempty = ',omitempty' in json_val

        if json_name == '-':
            # Hidden from JSON, use db field name for bson
            bson_name = db_val.split(',')[0]
        elif json_name == 'id':
            bson_name = '_id'
        else:
            bson_name = json_name

        bson_val = bson_name + (',omitempty' if omitempty else '')

        return f'json:"{json_val}" bson:"{bson_val}" db:"{db_val}"'

    content = re.sub(r'json:"([^"]*)" db:"([^"]*)"', replace_json_db, content)

    if content != original:
        with open(filepath, 'w') as f:
            f.write(content)
        # Count changes
        changes = len(re.findall(r'bson:"', content))
        print(f'Updated: {os.path.basename(filepath)} ({changes} bson tags added)')
    else:
        print(f'No changes: {os.path.basename(filepath)}')
