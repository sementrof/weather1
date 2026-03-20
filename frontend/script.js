const CREATE_USER_API_URL = '/create_user';
const WEATHER_API_URL = '/api/weather';

const STORAGE_DEVICE_ID_KEY = 'device_id';

function $id(id) {
    return document.getElementById(id);
}

async function fetchJson(url, options) {
    const response = await fetch(url, options);
    if (!response.ok) {
        const text = await response.text().catch(() => '');
        throw new Error(`HTTP ${response.status} ${text ? `- ${text}` : ''}`.trim());
    }
    return await response.json();
}

function loadDeviceId() {
    const raw = localStorage.getItem(STORAGE_DEVICE_ID_KEY);
    if (!raw) return null;
    const n = Number(raw);
    if (!Number.isFinite(n)) return null;
    return n;
}

function saveDeviceId(deviceId) {
    localStorage.setItem(STORAGE_DEVICE_ID_KEY, String(deviceId));
}

let state = {
    deviceId: null,
};

async function createOrUpdateDevice() {
    const name = $id('user-name').value.trim();
    const city = $id('device-city').value.trim();
    const infoEl = $id('device-info');

    if (!name || !city) {
        infoEl.textContent = 'Введите имя и город';
        return;
    }

    infoEl.textContent = 'Отправляем запрос в сервер...';

    const payload = { name, city };
    const data = await fetchJson(CREATE_USER_API_URL, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json; charset=utf-8' },
        body: JSON.stringify(payload),
    });

    state.deviceId = data.device_id;
    saveDeviceId(state.deviceId);
    infoEl.textContent = `Готово. device_id = ${state.deviceId}`;
}

function renderWeatherWidget(data) {
    const widget = $id('weather-widget');

    if (!data) {
        widget.innerHTML = '<p>Не удалось загрузить погоду.</p>';
        return;
    }

    const fromCacheText = data.from_cache ? 'Да (из кэша)' : 'Нет (обновлено)';
    const html = `
        <p>Город: ${data.city || '—'}</p>
        <p>Температура: ${data.temp !== undefined && data.temp !== null ? data.temp + '°C' : '—'}</p>
        <p>Состояние: ${data.condition || '—'}</p>
        <p>Источник: ${fromCacheText}</p>
    `;
    widget.innerHTML = html;
}

async function refreshWeather() {
    const widget = $id('weather-widget');
    widget.innerHTML = '<p>Загрузка погоды...</p>';

    try {
        const params = state.deviceId ? `?device_id=${encodeURIComponent(state.deviceId)}` : '';
        const data = await fetchJson(`${WEATHER_API_URL}${params}`);
        renderWeatherWidget(data);
    } catch (e) {
        console.error(e);
        $id('weather-widget').innerHTML = `<p>Не удалось загрузить погоду: ${e.message || e}</p>`;
    }
}

document.addEventListener('DOMContentLoaded', () => {
    state.deviceId = loadDeviceId();

    if (state.deviceId) {
        $id('device-info').textContent = `device_id из памяти: ${state.deviceId}`;
        refreshWeather();
    } else {
        $id('device-info').textContent = 'Сначала создайте устройство (город и имя)';
        renderWeatherWidget(null);
    }

    $id('create-user-btn').addEventListener('click', async () => {
        try {
            await createOrUpdateDevice();
            await refreshWeather();
        } catch (e) {
            console.error(e);
            $id('device-info').textContent = `Ошибка: ${e.message || e}`;
        }
    });

    $id('refresh-weather-btn').addEventListener('click', refreshWeather);
});