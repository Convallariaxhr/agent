// web/js/sse.js
//
// SSE client that reads a POST-based Server-Sent Events stream
// using the Fetch API + ReadableStream.

class SSEClient {
    constructor(url) {
        this.url = url;
        this.listeners = {};
        this.abortController = null;
    }

    /**
     * Connect to the SSE endpoint.
     * @param {object} body - JSON body sent as POST payload
     * @returns {Promise<void>}
     */
    connect(body) {
        this.abortController = new AbortController();

        return fetch(this.url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
            signal: this.abortController.signal,
        }).then(response => {
            if (!response.ok) {
                throw new Error(`SSE connection failed: ${response.status}`);
            }
            const reader = response.body.getReader();
            const decoder = new TextDecoder();
            let buffer = '';

            const pump = ({ done, value }) => {
                if (done) {
                    this.emit('disconnect', {});
                    return;
                }

                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split('\n');
                buffer = lines.pop() || '';

                let eventType = 'message';
                for (const line of lines) {
                    if (line.startsWith('event: ')) {
                        eventType = line.slice(7).trim();
                    } else if (line.startsWith('data: ')) {
                        const raw = line.slice(6);
                        let data;
                        try {
                            data = JSON.parse(raw);
                        } catch {
                            data = raw;
                        }
                        this.emit(eventType, data);
                    }
                    // ignore comments (lines starting with ':')
                }

                return reader.read().then(pump);
            };

            return reader.read().then(pump);
        });
    }

    /** Disconnect / abort the current request. */
    disconnect() {
        if (this.abortController) {
            this.abortController.abort();
            this.abortController = null;
        }
    }

    /**
     * Register an event listener.
     * @param {string} event - SSE event type ('token', 'done', 'session', 'error', etc.)
     * @param {function} callback
     */
    on(event, callback) {
        if (!this.listeners[event]) {
            this.listeners[event] = [];
        }
        this.listeners[event].push(callback);
    }

    /**
     * Emit an event to all registered listeners.
     * @param {string} event
     * @param {*} data
     */
    emit(event, data) {
        (this.listeners[event] || []).forEach(cb => {
            try {
                cb(data);
            } catch (e) {
                console.error(`SSE listener error [${event}]:`, e);
            }
        });
    }
}