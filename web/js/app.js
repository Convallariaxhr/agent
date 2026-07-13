// web/js/app.js
//
// Convallaria Web UI — main application logic.
// Handles chat messages, session management, SSE streaming, and panel toggling.

class ConvallariaApp {
    constructor() {
        this.currentSessionId = '';
        this.sessions = [];
        this.isStreaming = false;
        this.sse = null;

        this.init();
    }

    // ── Initialization ──────────────────────────────────

    init() {
        this.cacheElements();
        this.bindEvents();
        this.loadSessions();
    }

    cacheElements() {
        this.el = {
            chatMessages:  document.getElementById('chat-messages'),
            chatInput:     document.getElementById('chat-input'),
            inputWrapper:  document.getElementById('input-wrapper'),
            btnSend:       document.getElementById('btn-send'),
            btnNewChat:    document.getElementById('btn-new-chat'),
            sessionList:   document.getElementById('session-list'),
            filePanel:     document.getElementById('file-panel'),
            configPanel:   document.getElementById('config-panel'),
            btnToggleFiles:  document.getElementById('btn-toggle-files'),
            btnToggleConfig: document.getElementById('btn-toggle-config'),
            btnCloseFiles:   document.getElementById('btn-close-files'),
            btnCloseConfig:  document.getElementById('btn-close-config'),
        };
    }

    bindEvents() {
        // Send message
        this.el.btnSend.addEventListener('click', () => this.send());
        this.el.chatInput.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                this.send();
            }
        });

        // Auto-resize textarea
        this.el.chatInput.addEventListener('input', () => this.autoResize());

        // New chat
        this.el.btnNewChat.addEventListener('click', () => this.newChat());

        // Panel toggles
        this.el.btnToggleFiles.addEventListener('click', () => { this.togglePanel('file'); this.loadFiles(); });
        this.el.btnCloseFiles.addEventListener('click', () => this.togglePanel('file'));
        this.el.btnToggleConfig.addEventListener('click', () => { this.togglePanel('config'); this.loadConfig(); });
        this.el.btnCloseConfig.addEventListener('click', () => this.togglePanel('config'));
    }

    // ── Session Management ──────────────────────────────

    async loadSessions() {
        try {
            const resp = await fetch('/api/sessions');
            if (!resp.ok) return;
            this.sessions = await resp.json();
            this.renderSessions();
        } catch (e) {
            console.error('Failed to load sessions:', e);
        }
    }

    renderSessions() {
        if (!this.sessions.length) {
            this.el.sessionList.innerHTML = '';
            return;
        }
        this.el.sessionList.innerHTML = this.sessions.map(s =>
            `<div class="session-item${s.ID === this.currentSessionId ? ' active' : ''}"
                  data-id="${s.ID}"
                  onclick="app.switchSession('${s.ID}')"
                  oncontextmenu="app.showSessionMenu(event, '${s.ID}')">
                <span class="session-title">${this.escapeHtml(s.Title || 'Untitled')}</span>
            </div>`
        ).join('');

        // Close context menu on click outside
        if (!this._menuListener) {
            this._menuListener = () => this.hideSessionMenu();
            document.addEventListener('click', this._menuListener);
        }
    }

    showSessionMenu(e, id) {
        e.preventDefault();
        this.hideSessionMenu();
        const menu = document.createElement('div');
        menu.className = 'context-menu';
        menu.id = 'session-context-menu';
        menu.innerHTML = `<div class="context-menu-item" onclick="app.renameSession('${id}')">✏️ Rename</div>`;
        menu.style.left = e.clientX + 'px';
        menu.style.top = e.clientY + 'px';
        document.body.appendChild(menu);
        this._contextId = id;
    }

    hideSessionMenu() {
        const menu = document.getElementById('session-context-menu');
        if (menu) menu.remove();
    }

    renameSession(id) {
        this.hideSessionMenu();
        const item = document.querySelector(`.session-item[data-id="${id}"]`);
        if (!item) return;
        const titleSpan = item.querySelector('.session-title');
        const oldTitle = titleSpan.textContent;
        const input = document.createElement('input');
        input.className = 'session-rename-input';
        input.value = oldTitle === 'Untitled' ? '' : oldTitle;
        input.placeholder = 'Session name...';
        titleSpan.replaceWith(input);
        input.focus();
        input.select();

        const save = async () => {
            const newTitle = input.value.trim() || 'Untitled';
            try {
                await fetch(`/api/sessions/${id}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ title: newTitle }),
                });
            } catch (e) {
                console.error('Rename failed:', e);
            }
            const span = document.createElement('span');
            span.className = 'session-title';
            span.textContent = newTitle;
            input.replaceWith(span);
            this.loadSessions();
        };

        input.addEventListener('blur', save);
        input.addEventListener('keydown', (e) => {
            if (e.key === 'Enter') { e.preventDefault(); input.blur(); }
            if (e.key === 'Escape') { input.value = oldTitle; input.blur(); }
        });
    }

    switchSession(id) {
        this.currentSessionId = id;
        this.loadMessages(id);
        this.renderSessions();
    }

    newChat() {
        this.currentSessionId = '';
        this.el.chatMessages.innerHTML = `
            <div class="welcome">
                <div class="welcome-icon">
                    <svg width="64" height="64" viewBox="0 0 100 100" fill="none">
                        <defs>
                            <linearGradient id="newc-grad" x1="0" y1="0" x2="1" y2="1">
                                <stop offset="0%" stop-color="#bd9fff"/>
                                <stop offset="100%" stop-color="#a78bfa"/>
                            </linearGradient>
                        </defs>
                        <circle cx="50" cy="50" r="48" fill="url(#newc-grad)" opacity="0.15"/>
                        <circle cx="50" cy="50" r="46" fill="none" stroke="url(#newc-grad)" stroke-width="1.5" opacity="0.6"/>
                        <path d="M50 40 Q40 36 32 44 Q24 52 32 60 Q40 66 50 64 Q60 66 68 60 Q76 52 68 44 Q60 36 50 40Z" fill="white" opacity="0.95"/>
                        <circle cx="44" cy="50" r="3" fill="#2a2830"/>
                        <circle cx="45" cy="49" r="1.2" fill="white"/>
                        <circle cx="56" cy="50" r="3" fill="#2a2830"/>
                        <circle cx="57" cy="49" r="1.2" fill="white"/>
                        <ellipse cx="42" cy="56" rx="3" ry="1.5" fill="#bd9fff" opacity="0.5"/>
                        <ellipse cx="58" cy="56" rx="3" ry="1.5" fill="#bd9fff" opacity="0.5"/>
                        <circle cx="50" cy="38" r="3.5" fill="#bd9fff" opacity="0.6"/>
                    </svg>
                </div>
                <h2 class="welcome-title">What do you want to build?</h2>
                <p class="welcome-sub">Describe your task. I'll write, test, and fix the code.</p>
            </div>`;
        this.renderSessions();
    }

    async loadMessages(sessionId) {
        try {
            const resp = await fetch(`/api/sessions/${sessionId}`);
            if (!resp.ok) {
                console.error('loadMessages failed:', resp.status);
                this.appendMessage('system', `Failed to load messages (${resp.status})`);
                return;
            }
            const msgs = await resp.json();
            console.log('loadMessages got', msgs.length, 'messages for', sessionId);
            this.clearMessages();
            msgs.forEach(m => this.appendMessage(m.role, m.content));
        } catch (e) {
            console.error('Failed to load messages:', e);
            this.appendMessage('system', `Error: ${e.message}`);
        }
    }

    // ── Chat / SSE ──────────────────────────────────────

    async send() {
        const message = this.el.chatInput.value.trim();
        if (!message || this.isStreaming) return;

        this.el.chatInput.value = '';
        this.autoResize();

        this.clearWelcome();
        this.appendMessage('user', message);

        this.setStreaming(true);
        this.showThinking('正在思考...');

        this.sse = new SSEClient('/api/chat');
        let buffer = '';

        this.sse.on('session', (data) => {
            this.currentSessionId = data.id;
            this.loadSessions();
        });

        this.sse.on('token', (data) => {
            if (!buffer) this.showThinking('正在回复...');
            buffer += data.token;
            this.updateStreamingMessage(buffer);
        });

        this.sse.on('approval_required', (data) => {
            this.showApprovalDialog(data);
        });

        this.sse.on('error', (data) => {
            this.appendMessage('system', `Error: ${data.message}`);
        });

        this.sse.on('done', () => {
            this.finalizeStreamingMessage(buffer);
            this.setStreaming(false);
            this.loadSessions();
        });

        this.sse.on('disconnect', () => {
            this.setStreaming(false);
        });

        try {
            await this.sse.connect({
                session_id: this.currentSessionId,
                message: message,
            });
        } catch (e) {
            if (e.name !== 'AbortError') {
                this.appendMessage('system', `Connection error: ${e.message}`);
            }
            this.setStreaming(false);
        }
    }

    // ── Message Rendering ───────────────────────────────

    appendMessage(role, content) {
        const div = document.createElement('div');
        div.className = `message ${role}`;
        div.innerHTML = `<div class="message-bubble">${this.escapeHtml(content)}</div>`;
        this.el.chatMessages.appendChild(div);
        this.scrollToBottom();
    }

    updateStreamingMessage(content) {
        let el = this.el.chatMessages.querySelector('.message.streaming');
        if (!el) {
            el = document.createElement('div');
            el.className = 'message assistant streaming';
            el.innerHTML = '<div class="message-bubble"></div>';
            this.el.chatMessages.appendChild(el);
        }
        el.querySelector('.message-bubble').textContent = content;
        this.scrollToBottom();
    }

    finalizeStreamingMessage(content) {
        const el = this.el.chatMessages.querySelector('.message.streaming');
        if (el) {
            el.classList.remove('streaming');
            el.querySelector('.message-bubble').textContent = content;
        }
    }

    clearMessages() {
        this.el.chatMessages.innerHTML = '';
    }

    clearWelcome() {
        const welcome = this.el.chatMessages.querySelector('.welcome');
        if (welcome) welcome.remove();
    }

    scrollToBottom() {
        const el = this.el.chatMessages;
        el.scrollTop = el.scrollHeight;
    }

    // ── Streaming State ─────────────────────────────────

    setStreaming(active) {
        this.isStreaming = active;
        if (active) {
            this.el.inputWrapper.classList.add('streaming');
            this.el.btnSend.disabled = true;
        } else {
            this.el.inputWrapper.classList.remove('streaming');
            this.el.btnSend.disabled = false;
            this.el.chatInput.focus();
            this.hideThinking();
        }
    }

    showThinking(text) {
        this.hideThinking();
        const div = document.createElement('div');
        div.className = 'thinking-indicator';
        div.id = 'thinking-indicator';
        div.innerHTML = `
            <div class="thinking-spinner">
                <svg viewBox="0 0 100 100" fill="none">
                    <circle cx="50" cy="50" r="46" fill="none" stroke="#bd9fff" stroke-width="2" opacity="0.2"/>
                    <circle cx="50" cy="50" r="46" fill="none" stroke="#bd9fff" stroke-width="2"
                            stroke-dasharray="72 216" stroke-linecap="round" opacity="0.9"/>
                </svg>
            </div>
            <span class="thinking-text">${this.escapeHtml(text)}</span>`;
        this.el.chatMessages.appendChild(div);
        this.scrollToBottom();
    }

    hideThinking() {
        const el = document.getElementById('thinking-indicator');
        if (el) el.remove();
    }

    // ── Panel Toggle ────────────────────────────────────

    togglePanel(name) {
        const panel = name === 'file' ? this.el.filePanel : this.el.configPanel;
        const other  = name === 'file' ? this.el.configPanel : this.el.filePanel;
        const btn    = name === 'file' ? this.el.btnToggleFiles : this.el.btnToggleConfig;
        const otherBtn = name === 'file' ? this.el.btnToggleConfig : this.el.btnToggleFiles;

        if (panel.hasAttribute('hidden')) {
            panel.removeAttribute('hidden');
            other.setAttribute('hidden', '');
            btn.classList.add('active');
            otherBtn.classList.remove('active');
        } else {
            panel.setAttribute('hidden', '');
            btn.classList.remove('active');
        }
    }

    async loadFiles() {
        this.currentFileDir = this.currentFileDir || '';
        try {
            const resp = await fetch(`/api/files?dir=${encodeURIComponent(this.currentFileDir)}`);
            const entries = await resp.json();
            const tree = document.getElementById('file-tree');
            const breadcrumb = document.getElementById('file-breadcrumb');
            if (!entries || !entries.length) {
                tree.innerHTML = '<p class="empty-state">Empty directory</p>';
            } else {
                tree.innerHTML = entries.map(e =>
                    `<div class="file-entry ${e.isDir ? 'is-dir' : 'is-file'}"
                          data-name="${this.escapeHtml(e.name)}"
                          data-isdir="${e.isDir}"
                          onclick="app.navigateFile('${this.escapeHtml(e.name)}', ${e.isDir})">
                        ${e.isDir ? '📁' : '📄'} ${this.escapeHtml(e.name)}
                    </div>`
                ).join('');
            }
            breadcrumb.textContent = this.currentFileDir || '/';
        } catch (e) {
            console.error('Failed to load files:', e);
        }
    }

    navigateFile(name, isDir) {
        if (isDir) {
            this.currentFileDir = this.currentFileDir
                ? this.currentFileDir + '/' + name
                : name;
            this.loadFiles();
        } else {
            const filePath = this.currentFileDir
                ? this.currentFileDir + '/' + name
                : name;
            this.viewFile(filePath, name);
        }
    }

    goUpDir() {
        if (!this.currentFileDir) return;
        const parts = this.currentFileDir.split('/');
        parts.pop();
        this.currentFileDir = parts.length ? parts.join('/') : '';
        this.loadFiles();
    }

    async viewFile(path, name) {
        try {
            const resp = await fetch(`/api/files?path=${encodeURIComponent(path)}`);
            if (!resp.ok) {
                this.showToast('Cannot read file');
                return;
            }
            const content = await resp.text();
            const viewer = document.getElementById('file-viewer');
            const viewerTitle = document.getElementById('file-viewer-title');
            const viewerContent = document.getElementById('file-viewer-content');
            viewerTitle.textContent = name;
            viewerContent.textContent = content;
            viewer.removeAttribute('hidden');
        } catch (e) {
            console.error('Failed to read file:', e);
        }
    }

    closeFileViewer() {
        document.getElementById('file-viewer').setAttribute('hidden', '');
    }

    loadConfig() {
        const content = document.getElementById('config-content');
        content.innerHTML = `
            <div class="config-section">
                <label class="config-label">LLM Provider</label>
                <p class="config-value">deepseek</p>
            </div>
            <div class="config-section">
                <label class="config-label">Model</label>
                <p class="config-value">deepseek-chat</p>
            </div>
            <div class="config-section">
                <label class="config-label">Server Port</label>
                <p class="config-value">8080</p>
            </div>
            <div class="config-section">
                <label class="config-label">Database</label>
                <p class="config-value">convallaria.db (SQLite)</p>
            </div>
        `;
    }

    // ── Helpers ─────────────────────────────────────────

    autoResize() {
        const ta = this.el.chatInput;
        ta.style.height = 'auto';
        ta.style.height = Math.min(ta.scrollHeight, 120) + 'px';
    }

    escapeHtml(str) {
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    showToast(msg) {
        const toast = document.createElement('div');
        toast.textContent = msg;
        toast.style.cssText = 'position:fixed;bottom:24px;left:50%;transform:translateX(-50%);background:#bd9fff;color:#1c1b1f;padding:8px 20px;border-radius:8px;font-size:13px;font-weight:500;z-index:9999;pointer-events:none;transition:opacity 0.3s';
        document.body.appendChild(toast);
        setTimeout(() => { toast.style.opacity = '0'; setTimeout(() => toast.remove(), 300); }, 1500);
    }

    showApprovalDialog(data) {
        const overlay = document.getElementById('approval-overlay');
        const cmdEl = document.getElementById('approval-command');
        const reasonEl = document.getElementById('approval-reason');
        const denyBtn = document.getElementById('btn-deny');
        const allowBtn = document.getElementById('btn-allow-once');

        cmdEl.textContent = data.command || data.tool;
        reasonEl.textContent = data.reason || 'This action requires your approval.';
        overlay.removeAttribute('hidden');

        const respond = async (allowed) => {
            overlay.setAttribute('hidden', '');
            try {
                await fetch('/api/approve', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ id: data.id, allowed }),
                });
            } catch (e) {
                console.error('Approval error:', e);
            }
        };

        denyBtn.onclick = () => respond(false);
        allowBtn.onclick = () => respond(true);
    }
}

// Boot — robust initialization that works regardless of script load timing
(function boot() {
    try {
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', () => {
                window.app = new ConvallariaApp();
                console.log('Convallaria initialized (DOMContentLoaded)');
            });
        } else {
            window.app = new ConvallariaApp();
            console.log('Convallaria initialized (immediate)');
        }
    } catch (e) {
        console.error('Convallaria boot error:', e);
        document.body.insertAdjacentHTML('beforeend',
            '<div style="position:fixed;top:0;left:0;right:0;background:#f2b8b5;color:#1c1b1f;padding:12px;z-index:9999;font-family:sans-serif;">' +
            '<strong>Init Error:</strong> ' + e.message + '</div>');
    }
})();