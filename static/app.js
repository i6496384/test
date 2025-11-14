// Глобальные переменные
let currentServer = null;

// Создание нового сервера
function createNewServer() {
    console.log('Creating new server...');
    
    // Проверяем, что мы на правильной странице
    if (window.location.pathname !== '/dashboard') {
        showAlert('Функция доступна только на странице dashboard', 'warning');
        return;
    }
    
    currentServer = null; // Сбрасываем текущий сервер
    
    // Проверяем существование формы
    const serverForm = document.getElementById('serverForm');
    if (!serverForm) {
        console.error('Server form not found');
        showAlert('Форма сервера не найдена', 'danger');
        return;
    }
    
    // Очищаем форму
    serverForm.reset();
    
    // Устанавливаем значения по умолчанию
    const elements = {
        serverPort: document.getElementById('serverPort'),
        serverNetwork: document.getElementById('serverNetwork'),
        serverDNS: document.getElementById('serverDNS'),
        serverAllowedIPs: document.getElementById('serverAllowedIPs')
    };
    
    if (elements.serverPort) elements.serverPort.value = 51820;
    if (elements.serverNetwork) elements.serverNetwork.value = '10.0.0.0/24';
    if (elements.serverDNS) elements.serverDNS.value = '8.8.8.8';
    if (elements.serverAllowedIPs) elements.serverAllowedIPs.value = '0.0.0.0/0';
    
    // Скрываем информацию о сервере
    const serverInfo = document.getElementById('serverInfo');
    if (serverInfo) {
        serverInfo.style.display = 'none';
    }
    
    // Показываем уведомление
    showAlert('Форма очищена. Готов к созданию нового сервера', 'info');
}

// Инициализация при загрузке страницы
document.addEventListener('DOMContentLoaded', function() {
    console.log('DOMContentLoaded fired');
    console.log('Current path:', window.location.pathname);
    
    // Проверяем, на какой странице мы находимся
    const currentPath = window.location.pathname;
    
    // Проверяем загрузку CSS
    const cssLinks = document.querySelectorAll('link[rel="stylesheet"]');
    console.log('CSS files loaded:', cssLinks.length);
    cssLinks.forEach(link => console.log('CSS href:', link.href));
    
    if (currentPath === '/dashboard') {
        console.log('On dashboard page, initializing...');
        
        // Только на странице dashboard загружаем данные и формы
        loadServerConfig();
        loadClients();
        loadStats();
        
        // Обработчики форм (только если элементы существуют)
        const serverForm = document.getElementById('serverForm');
        if (serverForm) {
            serverForm.addEventListener('submit', handleServerSubmit);
        }
        
        const clientForm = document.getElementById('clientForm');
        if (clientForm) {
            clientForm.addEventListener('submit', handleClientSubmit);
        }
    }
});

// Загрузка конфигурации сервера
async function loadServerConfig() {
    try {
        const response = await fetch('/api/server');
        const data = await response.json();
        
        if (data.success && data.data.id) {
            currentServer = data.data;
            populateServerForm(data.data);
            showServerInfo(data.data);
        }
    } catch (error) {
        console.error('Ошибка загрузки сервера:', error);
    }
}

// Заполнение формы сервера
function populateServerForm(server) {
    const elements = {
        serverName: document.getElementById('serverName'),
        serverPort: document.getElementById('serverPort'),
        serverNetwork: document.getElementById('serverNetwork'),
        serverDNS: document.getElementById('serverDNS'),
        serverEndpoint: document.getElementById('serverEndpoint'),
        serverAllowedIPs: document.getElementById('serverAllowedIPs')
    };
    
    // Проверяем существование каждого элемента перед установкой значения
    if (elements.serverName) {
        elements.serverName.value = server.name || '';
    }
    if (elements.serverPort) {
        elements.serverPort.value = server.listen_port || 51820;
    }
    if (elements.serverNetwork) {
        elements.serverNetwork.value = server.network || '10.0.0.0/24';
    }
    if (elements.serverDNS) {
        elements.serverDNS.value = server.dns || '8.8.8.8';
    }
    if (elements.serverEndpoint) {
        elements.serverEndpoint.value = server.endpoint || '';
    }
    if (elements.serverAllowedIPs) {
        elements.serverAllowedIPs.value = server.allowed_ips || '0.0.0.0/0';
    }
}

// Отображение информации о сервере
function showServerInfo(server) {
    const elements = {
        currentServerId: document.getElementById('currentServerId'),
        currentServerPort: document.getElementById('currentServerPort'),
        currentServerNetwork: document.getElementById('currentServerNetwork'),
        currentServerEndpoint: document.getElementById('currentServerEndpoint'),
        serverInfo: document.getElementById('serverInfo')
    };
    
    // Проверяем существование каждого элемента перед установкой текста
    if (elements.currentServerId) {
        elements.currentServerId.textContent = server.id || 'Не задан';
    }
    if (elements.currentServerPort) {
        elements.currentServerPort.textContent = server.listen_port || 'Не задан';
    }
    if (elements.currentServerNetwork) {
        elements.currentServerNetwork.textContent = server.network || 'Не задана';
    }
    if (elements.currentServerEndpoint) {
        elements.currentServerEndpoint.textContent = server.endpoint || 'Не задан';
    }
    if (elements.serverInfo) {
        elements.serverInfo.style.display = 'block';
    }
}

// Обработка отправки формы сервера
async function handleServerSubmit(event) {
    event.preventDefault();
    
    const formData = {
        name: document.getElementById('serverName').value,
        listen_port: parseInt(document.getElementById('serverPort').value),
        network: document.getElementById('serverNetwork').value,
        dns: document.getElementById('serverDNS').value,
        endpoint: document.getElementById('serverEndpoint').value,
        allowed_ips: document.getElementById('serverAllowedIPs').value
    };
    
    try {
        const method = currentServer ? 'PUT' : 'POST';
        const url = currentServer ? `/api/server/${currentServer.id}` : '/api/server';
        const action = currentServer ? 'обновлен' : 'создан';
        
        const response = await fetch(url, {
            method: method,
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(formData)
        });
        
        const data = await response.json();
        
        if (data.success) {
            showAlert(`Сервер успешно ${action}`, 'success');
            currentServer = data.data;
            showServerInfo(data.data);
        } else {
            showAlert('Ошибка: ' + data.error, 'danger');
        }
    } catch (error) {
        console.error('Ошибка сохранения сервера:', error);
        showAlert('Ошибка сохранения сервера', 'danger');
    }
}

// Обработка отправки формы клиента
async function handleClientSubmit(event) {
    event.preventDefault();
    
    if (!currentServer) {
        showAlert('Сначала настройте сервер', 'warning');
        return;
    }
    
    const formData = {
        server_id: currentServer.id,
        name: document.getElementById('clientName').value,
        email: document.getElementById('clientEmail').value
    };
    
    try {
        const response = await fetch('/api/clients', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(formData)
        });
        
        const data = await response.json();
        
        if (data.success) {
            showAlert('Клиент добавлен', 'success');
            document.getElementById('clientForm').reset();
            loadClients();
            loadStats();
        } else {
            showAlert('Ошибка: ' + data.error, 'danger');
        }
    } catch (error) {
        console.error('Ошибка добавления клиента:', error);
        showAlert('Ошибка добавления клиента', 'danger');
    }
}

// Загрузка списка клиентов
async function loadClients() {
    console.log('loadClients called, current path:', window.location.pathname);
    
    // Проверяем, что мы на правильной странице
    if (window.location.pathname !== '/dashboard') {
        console.log('Skipping loadClients - not on dashboard page');
        return;
    }
    
    try {
        const serverId = currentServer ? currentServer.id : '';
        const url = serverId ? `/api/clients?server_id=${serverId}` : '/api/clients';
        
        console.log('Fetching clients from:', url);
        const response = await fetch(url);
        const data = await response.json();
        
        console.log('Clients API response:', data);
        
        if (data.success) {
            renderClients(data.data);
        }
    } catch (error) {
        console.error('Ошибка загрузки клиентов:', error);
        renderClients([]);
    }
}

// Отображение списка клиентов
function renderClients(clients) {
    console.log('renderClients called with:', clients.length, 'clients');
    console.log('Current path:', window.location.pathname);
    
    const tbody = document.getElementById('clientsTableBody');
    console.log('tbody element:', tbody);
    console.log('All tbody elements:', document.querySelectorAll('tbody'));
    console.log('All table elements:', document.querySelectorAll('table'));
    console.log('All elements with id:', document.querySelectorAll('[id]'));
    
    // Проверяем, существует ли элемент
    if (!tbody) {
        console.error('Element clientsTableBody not found!');
        console.log('Document ready state:', document.readyState);
        console.log('Body innerHTML length:', document.body.innerHTML.length);
        
        // Проверяем, может ли быть проблема с видимостью элементов
        const table = document.querySelector('table');
        if (table) {
            console.log('Table found, style:', window.getComputedStyle(table));
        }
        
        return;
    }
    
    console.log('Element found, proceeding to render...');
    
    if (clients.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="text-center">Клиенты не найдены</td></tr>';
        return;
    }
    
    tbody.innerHTML = clients.map(client => `
        <tr>
            <td>${client.name}</td>
            <td>${client.email || '-'}</td>
            <td>${client.allowed_ips}</td>
            <td>
                <span class="status-badge ${getStatusClass(client)}">
                    ${getStatusText(client)}
                </span>
            </td>
            <td>
                ${client.downloaded ? '<span class="text-success">✓</span>' : '<span class="text-muted">✗</span>'}
            </td>
            <td>
                <button class="btn btn-primary" onclick="downloadConfig('${client.id}')" title="Скачать конфиг">
                    Скачать
                </button>
                <button class="btn btn-warning" onclick="toggleClient('${client.id}', ${client.is_disabled})" title="${client.is_disabled ? 'Включить' : 'Отключить'}">
                    ${client.is_disabled ? 'Включить' : 'Отключить'}
                </button>
                <button class="btn btn-danger" onclick="deleteClient('${client.id}')" title="Удалить">
                    Удалить
                </button>
            </td>
        </tr>
    `).join('');
}

// Получение CSS класса для статуса
function getStatusClass(client) {
    if (client.is_disabled) return 'status-disabled';
    if (client.downloaded) return 'status-downloaded';
    return 'status-active';
}

// Получение текста статуса
function getStatusText(client) {
    if (client.is_disabled) return 'Отключен';
    if (client.downloaded) return 'Активен (скачал)';
    return 'Активен';
}

// Скачивание конфигурации клиента
function downloadConfig(clientId) {
    window.open(`/api/clients/${clientId}/config`, '_blank');
}

// Переключение статуса клиента
async function toggleClient(clientId, isDisabled) {
    try {
        const action = isDisabled ? 'enable' : 'disable';
        const response = await fetch(`/api/clients/${clientId}/${action}`, {
            method: 'PUT'
        });
        
        const data = await response.json();
        
        if (data.success) {
            showAlert(`Клиент ${isDisabled ? 'включен' : 'отключен'}`, 'success');
            loadClients();
            loadStats();
        } else {
            showAlert('Ошибка: ' + data.error, 'danger');
        }
    } catch (error) {
        console.error('Ошибка переключения клиента:', error);
        showAlert('Ошибка изменения статуса клиента', 'danger');
    }
}

// Удаление клиента
async function deleteClient(clientId) {
    if (!confirm('Вы уверены, что хотите удалить этого клиента?')) {
        return;
    }
    
    try {
        const response = await fetch(`/api/clients/${clientId}`, {
            method: 'DELETE'
        });
        
        const data = await response.json();
        
        if (data.success) {
            showAlert('Клиент удален', 'success');
            loadClients();
            loadStats();
        } else {
            showAlert('Ошибка: ' + data.error, 'danger');
        }
    } catch (error) {
        console.error('Ошибка удаления клиента:', error);
        showAlert('Ошибка удаления клиента', 'danger');
    }
}

// Загрузка статистики
async function loadStats() {
    try {
        const response = await fetch('/api/stats');
        const data = await response.json();
        
        if (data.success) {
            updateStatsDisplay(data.data);
        }
    } catch (error) {
        console.error('Ошибка загрузки статистики:', error);
    }
}

// Обновление отображения статистики
function updateStatsDisplay(stats) {
    const elements = {
        totalClients: document.getElementById('totalClients'),
        activeClients: document.getElementById('activeClients'),
        disabledClients: document.getElementById('disabledClients'),
        downloadedCount: document.getElementById('downloadedCount')
    };
    
    // Проверяем существование каждого элемента перед установкой текста
    if (elements.totalClients) {
        elements.totalClients.textContent = stats.total_clients;
    }
    if (elements.activeClients) {
        elements.activeClients.textContent = stats.active_clients;
    }
    if (elements.disabledClients) {
        elements.disabledClients.textContent = stats.disabled_clients;
    }
    if (elements.downloadedCount) {
        elements.downloadedCount.textContent = stats.downloaded_count;
    }
}

// Обновление статистики
function refreshStats() {
    // Проверяем, что мы на странице dashboard
    if (window.location.pathname === '/dashboard') {
        loadStats();
        loadClients();
        showAlert('Данные обновлены', 'info');
    } else {
        showAlert('Обновление доступно только на странице dashboard', 'warning');
    }
}

// Показ уведомления
function showAlert(message, type) {
    // Создаем элемент уведомления с кастомными классами
    const alertDiv = document.createElement('div');
    alertDiv.className = `alert alert-${type}`;
    alertDiv.style.cssText = 'position: fixed; top: 20px; right: 20px; z-index: 9999; min-width: 300px; max-width: 400px;';
    alertDiv.innerHTML = `
        ${message}
        <button type="button" onclick="this.parentElement.remove()" style="float: right; margin-left: 10px; background: none; border: none; cursor: pointer;">&times;</button>
    `;
    
    document.body.appendChild(alertDiv);
    
    // Автоматически скрываем через 3 секунды
    setTimeout(() => {
        if (alertDiv.parentNode) {
            alertDiv.parentNode.removeChild(alertDiv);
        }
    }, 3000);
}