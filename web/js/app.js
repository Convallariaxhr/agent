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
        this.el.btnToggleFiles.addEventListener('click', () => this.togglePanel('file'));
        this.el.btnCloseFiles.addEventListener('click', () => this.togglePanel('file'));
        this.el.btnToggleConfig.addEventListener('click', () => this.togglePanel('config'));
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
                  onclick="app.switchSession('${s.ID}')">
                ${this.escapeHtml(s.Title || 'Untitled')}
            </div>`
        ).join('');
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
                    <svg width="48" height="48" viewBox="0 0 48 48" fill="none">
                        <circle cx="24" cy="24" r="8" stroke="currentColor" stroke-width="1.5"/>
                        <path d="M24 8v4M24 36v4M8 24h4M36 24h4" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
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
            if (!resp.ok) return;
            const msgs = await resp.json();
            this.clearMessages();
            msgs.forEach(m => this.appendMessage(m.Role, m.Content));
        } catch (e) {
            console.error('Failed to load messages:', e);
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

        this.sse = new SSEClient('/api/chat');
        let buffer = '';

        this.sse.on('session', (data) => {
            this.currentSessionId = data.id;
            this.loadSessions();
        });

        this.sse.on('token', (data) => {
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
        }
    }

    // ── Panel Toggle ────────────────────────────────────

    togglePanel(name) {
        const panel = name === 'file' ? this.el.filePanel : this.el.configPanel;
        const other  = name === 'file' ? this.el.configPanel : this.el.filePanel;

        if (panel.hasAttribute('hidden')) {
            panel.removeAttribute('hidden');
            other.setAttribute('hidden', '');
        } else {
            panel.setAttribute('hidden', '');
        }
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