/**
 * 聊天室客户端脚本
 */

// 全局变量
let socket = null;
let currentUsername = '';
let currentChannel = 'general';
let reconnectAttempts = 0;
let reconnectInterval = null;

// DOM元素
const usernameInput = document.getElementById('username');
const loginBtn = document.getElementById('login-btn');
const logoutBtn = document.getElementById('logout-btn');
const userProfile = document.getElementById('user-profile');
const loggedUser = document.getElementById('logged-user');
const currentUsernameSpan = document.getElementById('current-username');
const channelList = document.getElementById('channel-list');
const currentChannelHeader = document.getElementById('current-channel');
const messageContainer = document.getElementById('message-container');
const messageInput = document.getElementById('message-input');
const sendBtn = document.getElementById('send-btn');
const newChannelInput = document.getElementById('new-channel');
const addChannelBtn = document.getElementById('add-channel-btn');
const userList = document.getElementById('user-list');

/**
 * 初始化聊天室
 */
function init() {
    // 绑定事件监听器
    loginBtn.addEventListener('click', connectToChat);
    logoutBtn.addEventListener('click', disconnectFromChat);
    sendBtn.addEventListener('click', sendMessage);
    messageInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage();
        }
    });

    // 频道点击事件
    channelList.addEventListener('click', (e) => {
        if (e.target.tagName === 'LI') {
            changeChannel(e.target.getAttribute('data-channel'));
        }
    });

    // 添加新频道
    addChannelBtn.addEventListener('click', addNewChannel);
    newChannelInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            addNewChannel();
        }
    });

    // 检查本地存储中的用户名
    const savedUsername = localStorage.getItem('chatUsername');
    if (savedUsername) {
        usernameInput.value = savedUsername;
        connectToChat();
    }
}

/**
 * 连接到聊天服务器
 */
function connectToChat() {
    const username = usernameInput.value.trim();
    if (!username) {
        alert('请输入用户名');
        return;
    }

    currentUsername = username;
    localStorage.setItem('chatUsername', username);

    // 更新UI
    userProfile.style.display = 'none';
    loggedUser.style.display = 'block';
    currentUsernameSpan.textContent = username;
    messageInput.disabled = false;
    sendBtn.disabled = false;

    // 连接WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws?username=${encodeURIComponent(username)}`;

    socket = new WebSocket(wsUrl);

    socket.onopen = () => {
        console.log('WebSocket连接已建立');
        showSystemMessage('已连接到聊天服务器');

        // 重置重连计数器
        reconnectAttempts = 0;
        if (reconnectInterval) {
            clearInterval(reconnectInterval);
            reconnectInterval = null;
        }

        // 加载历史消息
        loadHistoryMessages(currentChannel);

        // 订阅当前频道
        subscribeToChannel(currentChannel);
    };

    socket.onmessage = (event) => {
        const message = JSON.parse(event.data);
        displayMessage(message);
    };

    socket.onclose = (event) => {
        console.log('WebSocket连接已关闭', event);

        if (currentUsername) {
            showSystemMessage('与服务器的连接已断开，正在尝试重新连接...');

            // 尝试重新连接
            if (!reconnectInterval) {
                reconnectInterval = setInterval(() => {
                    if (reconnectAttempts < 5) {
                        reconnectAttempts++;
                        console.log(`尝试重新连接 (${reconnectAttempts}/5)...`);
                        connectToChat();
                    } else {
                        clearInterval(reconnectInterval);
                        reconnectInterval = null;
                        showSystemMessage('无法连接到服务器，请稍后再试');
                        disconnectFromChat(true);
                    }
                }, 5000);
            }
        }
    };

    socket.onerror = (error) => {
        console.error('WebSocket错误:', error);
        showSystemMessage('连接错误，请稍后再试');
    };
}

/**
 * 断开与聊天服务器的连接
 * @param {boolean} skipSocketClose - 是否跳过关闭socket（在连接已断开的情况下）
 */
function disconnectFromChat(skipSocketClose = false) {
    if (socket && !skipSocketClose) {
        socket.close();
    }

    // 更新UI
    userProfile.style.display = 'block';
    loggedUser.style.display = 'none';
    messageInput.disabled = true;
    sendBtn.disabled = true;

    // 清除状态
    currentUsername = '';
    messageContainer.innerHTML = '';

    // 停止重连尝试
    if (reconnectInterval) {
        clearInterval(reconnectInterval);
        reconnectInterval = null;
    }
}

/**
 * 发送消息
 */
function sendMessage() {
    if (!socket || socket.readyState !== WebSocket.OPEN) {
        showSystemMessage('未连接到服务器，无法发送消息');
        return;
    }

    const content = messageInput.value.trim();
    if (!content) return;

    const message = {
        type: 'message',
        content: content,
        channel: currentChannel,
        username: currentUsername,
        created_at: new Date().toISOString()
    };

    socket.send(JSON.stringify(message));
    messageInput.value = '';
}

/**
 * 显示消息
 * @param {Object} message - 消息对象
 */
function displayMessage(message) {
    const messageElement = document.createElement('div');
    messageElement.className = 'message';

    if (message.type === 'system') {
        messageElement.classList.add('system-message');
        messageElement.innerHTML = `<div class="message-content">${message.content}</div>`;
    } else {
        const isCurrentUser = message.username === currentUsername;
        if (isCurrentUser) {
            messageElement.classList.add('my-message');
        }

        const time = new Date(message.created_at).toLocaleTimeString();
        messageElement.innerHTML = `
            <div class="message-header">
                <span class="message-username">${message.username}</span>
                <span class="message-time">${time}</span>
            </div>
            <div class="message-content">${message.content}</div>
        `;
    }

    messageContainer.appendChild(messageElement);
    messageContainer.scrollTop = messageContainer.scrollHeight;
}

/**
 * 显示系统消息
 * @param {string} content - 消息内容
 */
function showSystemMessage(content) {
    const message = {
        type: 'system',
        content: content,
        created_at: new Date().toISOString()
    };
    displayMessage(message);
}

/**
 * 加载历史消息
 * @param {string} channel - 频道名称
 */
function loadHistoryMessages(channel) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;

    const request = {
        type: 'history',
        channel: channel
    };

    socket.send(JSON.stringify(request));
}

/**
 * 订阅频道
 * @param {string} channel - 频道名称
 */
function subscribeToChannel(channel) {
    if (!socket || socket.readyState !== WebSocket.OPEN) return;

    const request = {
        type: 'subscribe',
        channel: channel
    };

    socket.send(JSON.stringify(request));
}

/**
 * 切换频道
 * @param {string} channel - 频道名称
 */
function changeChannel(channel) {
    if (channel === currentChannel) return;

    // 更新UI
    const channelItems = channelList.querySelectorAll('li');
    channelItems.forEach(item => {
        if (item.getAttribute('data-channel') === channel) {
            item.classList.add('active');
        } else {
            item.classList.remove('active');
        }
    });

    currentChannel = channel;
    currentChannelHeader.textContent = getChannelDisplayName(channel);
    messageContainer.innerHTML = '';

    // 加载新频道的历史消息
    loadHistoryMessages(channel);

    // 订阅新频道
    subscribeToChannel(channel);
}

/**
 * 添加新频道
 */
function addNewChannel() {
    const channelName = newChannelInput.value.trim();
    if (!channelName) return;

    // 检查频道是否已存在
    const existingChannel = document.querySelector(`#channel-list li[data-channel="${channelName}"]`);
    if (existingChannel) {
        alert('该频道已存在');
        return;
    }

    // 创建新频道
    const channelItem = document.createElement('li');
    channelItem.setAttribute('data-channel', channelName);
    channelItem.textContent = channelName;
    channelList.appendChild(channelItem);

    // 清空输入框
    newChannelInput.value = '';

    // 切换到新频道
    changeChannel(channelName);
}

/**
 * 获取频道显示名称
 * @param {string} channel - 频道名称
 * @returns {string} 频道显示名称
 */
function getChannelDisplayName(channel) {
    const channelMap = {
        'general': '综合频道',
        'tech': '技术讨论',
        'random': '随便聊聊'
    };

    return channelMap[channel] || channel;
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', init);