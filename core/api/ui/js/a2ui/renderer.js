import { componentCatalog } from './catalog.js';

export class A2UIRenderer {
    constructor(container, streamUrl, actionUrl) {
        this.container = container;
        this.streamUrl = streamUrl;
        this.actionUrl = actionUrl;
        this.components = new Map();
        this.eventSource = null;
    }

    start() {
        this.eventSource = new EventSource(this.streamUrl);
        this.eventSource.onmessage = (event) => {
            const data = JSON.parse(event.data);
            this.updateState(data);
            this.render();
        };
        this.eventSource.onerror = (err) => {
            console.error("SSE Error:", err);
        };
    }

    updateState(components) {
        this.components.clear();
        components.forEach(c => {
            this.components.set(c.id, c);
        });
    }

    async dispatch(componentId, actionType, data = {}) {
        try {
            await fetch(this.actionUrl, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ componentId, actionType, data }),
                credentials: 'include'
            });
        } catch (err) {
            console.error("Action Error:", err);
        }
    }

    render() {
        this.container.innerHTML = '';
        // Start rendering from the 'root' component
        const root = this.renderComponent('root');
        if (root) {
            this.container.appendChild(root);
        }
    }

    renderComponent(id) {
        const config = this.components.get(id);
        if (!config) return null;

        const renderFunc = componentCatalog[config.type];
        if (!renderFunc) {
            console.warn(`Unknown component type: ${config.type}`);
            return null;
        }

        const children = (config.children || []).map(cid => this.renderComponent(cid)).filter(c => c !== null);
        
        return renderFunc(
            config.attributes || {}, 
            children, 
            (actionType, data) => this.dispatch(id, actionType, data)
        );
    }
}
