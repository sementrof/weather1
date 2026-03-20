#!/bin/bash

BASE_URL="http://localhost:8080"
PASS=0
FAIL=0

green="\033[0;32m"
red="\033[0;31m"
yellow="\033[0;33m"
reset="\033[0m"

check() {
    local test_name="$1"
    local expected="$2"
    local actual="$3"

    if [ "$actual" = "$expected" ]; then
        echo -e "${green}✅ PASS${reset} — $test_name (код: $actual)"
        PASS=$((PASS + 1))
    else
        echo -e "${red}❌ FAIL${reset} — $test_name (ожидалось: $expected, получено: $actual)"
        FAIL=$((FAIL + 1))
    fi
}

echo "================================================"
echo "   ТЕСТИРОВАНИЕ БЕЗОПАСНОСТИ — weather1"
echo "================================================"
echo ""

# ── БЛОК 1: Базовая работа 
echo -e "${yellow}БЛОК 1 — Базовая работа${reset}"
echo "------------------------------------------------"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/create_user" \
  -H "Content-Type: application/json" \
  -d '{"name":"Ivan","city":"Sochi"}')
check "Создание устройства" "201" "$CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/weather?device_id=1")
check "Получение погоды по device_id=1" "200" "$CODE"

BODY=$(curl -s "$BASE_URL/api/weather?device_id=1")
if echo "$BODY" | grep -q "from_cache\":true"; then
    echo -e "${green}✅ PASS${reset} — Кэш работает (from_cache: true при повторном запросе)"
    PASS=$((PASS + 1))
else
    echo -e "${red}❌ FAIL${reset} — Кэш не работает"
    FAIL=$((FAIL + 1))
fi

echo ""

# ── БЛОК 2: Несанкционированный доступ 
echo -e "${yellow}БЛОК 2 — Несанкционированный доступ${reset}"
echo "------------------------------------------------"

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/weather")
check "Запрос без device_id → 401" "401" "$CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/weather?device_id=999999")
check "Несуществующий device_id=999999 → 404" "404" "$CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/weather?device_id=-1")
check "Отрицательный device_id=-1 → 404" "404" "$CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/weather?device_id=0")
check "Нулевой device_id=0 → 404" "404" "$CODE"

echo ""

# ── БЛОК 3: SQL-инъекции 
echo -e "${yellow}БЛОК 3 — SQL-инъекции${reset}"
echo "------------------------------------------------"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/create_user" \
  -H "Content-Type: application/json" \
  -d '{"name":"'"'"'; DROP TABLE user; --","city":"Moscow"}')
check "Инъекция DROP TABLE в name → 201" "201" "$CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/create_user" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","city":"'"'"' OR '"'"'1'"'"'='"'"'1"}')
check "Инъекция OR 1=1 в city → 201" "201" "$CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  "$BASE_URL/api/weather?device_id=1%20OR%201%3D1")
check "Инъекция в device_id (OR 1=1) → 400" "400" "$CODE"

START=$(date +%s)
CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  "$BASE_URL/api/weather?device_id=1%3BSELECT%2Bpg_sleep%285%29--")
END=$(date +%s)
ELAPSED=$((END - START))
check "Time-based инъекция в device_id → 400" "400" "$CODE"
if [ $ELAPSED -lt 5 ]; then
    echo -e "\033[0;32m✅ PASS\033[0m — Сервер не завис (ответ за ${ELAPSED}с, не 5с)"
else
    echo -e "\033[0;31m❌ FAIL\033[0m — Возможна time-based инъекция (ответ за ${ELAPSED}с)"
fi

# Проверить что таблица жива после инъекций
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/create_user" \
  -H "Content-Type: application/json" \
  -d '{"name":"AfterInjection","city":"Moscow"}')
check "БД жива после инъекций → 201" "201" "$CODE"

echo ""

# ── БЛОК 4: Валидация полей 
echo -e "${yellow}БЛОК 4 — Валидация полей${reset}"
echo "------------------------------------------------"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/create_user" \
  -H "Content-Type: application/json" \
  -d '{"name":"","city":""}')
check "Пустые поля → 400" "400" "$CODE"

LONG_NAME=$(python3 -c "print('a'*101)")
CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/create_user" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$LONG_NAME\",\"city\":\"Moscow\"}")
check "Имя длиннее 100 символов → 400" "400" "$CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/create_user" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","city":"'"$(python3 -c "print('a'*101)")"'"}')
check "Город длиннее 100 символов → 400" "400" "$CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/create_user" \
  -H "Content-Type: application/json" \
  -d '{name: Ivan}')
check "Сломанный JSON → 400" "400" "$CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/create_user" \
  -H "Content-Type: text/plain" \
  -d 'name=Ivan&city=Moscow')
check "Неверный Content-Type → 400" "400" "$CODE"

echo ""

# ── БЛОК 5: Неподдерживаемые методы
echo -e "${yellow}БЛОК 5 — Неподдерживаемые методы${reset}"
echo "------------------------------------------------"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$BASE_URL/api/weather")
check "DELETE /api/weather → 405" "405" "$CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/create_user" \
  -H "Content-Type: application/json" \
  -d '{"name":"x","city":"y"}')
check "PUT /create_user → 405" "405" "$CODE"

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/admin")
check "Несуществующий эндпоинт /api/admin → 404" "404" "$CODE"

echo ""

# ── ИТОГ ─────────────────────────────────────────────────────────────────────
echo "================================================"
echo -e "   ИТОГ: ${green}✅ PASS: $PASS${reset} | ${red}❌ FAIL: $FAIL${reset}"
echo "================================================"