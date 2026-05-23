#!/bin/bash
set -euo pipefail

API="http://localhost:8080/api/v1"
COMPETITOR="${1:-Cursor}"

echo "🚀 CompetifyAI Demo — 分析竞品: $COMPETITOR"
echo ""

# 1. 创建任务
echo "[1/4] 创建分析任务..."
TASK_RES=$(curl -s -X POST "$API/tasks" \
  -H "Content-Type: application/json" \
  -d "{\"competitor_name\":\"$COMPETITOR\",\"dimensions\":[\"feature\",\"pricing\",\"tech\"],\"priority\":3,\"requested_by\":\"demo\"}")
TASK_ID=$(echo "$TASK_RES" | grep -o '"task_id":"[^"]*"' | cut -d'"' -f4)
echo "      task_id = $TASK_ID"

# 2. 轮询状态
echo "[2/4] 轮询任务状态..."
for i in {1..30}; do
  STATUS=$(curl -s "$API/tasks/$TASK_ID" | grep -o '"status":"[^"]*"' | cut -d'"' -f4 || echo "unknown")
  echo "      status = $STATUS"
  if [ "$STATUS" = "done" ]; then
    break
  fi
  sleep 2
done

# 3. 获取报告
echo "[3/4] 获取分析报告..."
REPORT=$(curl -s "$API/reports/$TASK_ID")
echo "$REPORT" | python3 -m json.tool 2>/dev/null || echo "$REPORT"

# 4. 验证 Merkle
echo "[4/4] 验证 Merkle Root..."
VERIFY=$(curl -s "$API/audit/$TASK_ID/verify")
echo "      $VERIFY"

echo ""
echo "✅ Demo 完成"
