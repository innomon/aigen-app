export const componentCatalog = {
    "Column": (attributes, children) => {
        const div = document.createElement('div');
        div.className = 'd-flex flex-column gap-3';
        children.forEach(child => div.appendChild(child));
        return div;
    },
    "Row": (attributes, children) => {
        const div = document.createElement('div');
        div.className = 'd-flex flex-row gap-3';
        children.forEach(child => div.appendChild(child));
        return div;
    },
    "Card": (attributes, children) => {
        const card = document.createElement('div');
        card.className = 'card';
        const body = document.createElement('div');
        body.className = 'card-body';
        children.forEach(child => body.appendChild(child));
        card.appendChild(body);
        return card;
    },
    "Heading": (attributes) => {
        const h = document.createElement(`h${attributes.level || 1}`);
        h.textContent = attributes.content;
        return h;
    },
    "Text": (attributes) => {
        const p = document.createElement('p');
        p.textContent = attributes.content;
        return p;
    },
    "Button": (attributes, children, dispatch) => {
        const btn = document.createElement('button');
        btn.className = `btn btn-${attributes.variant || 'primary'}`;
        btn.textContent = attributes.label;
        if (attributes.action) {
            btn.addEventListener('click', () => {
                dispatch(attributes.action);
            });
        }
        return btn;
    },
    "TextField": (attributes, children, dispatch) => {
        const group = document.createElement('div');
        group.className = 'mb-3';
        if (attributes.label) {
            const label = document.createElement('label');
            label.className = 'form-label';
            label.textContent = attributes.label;
            group.appendChild(label);
        }
        const input = document.createElement('input');
        input.type = 'text';
        input.className = 'form-control';
        input.placeholder = attributes.placeholder || '';
        input.addEventListener('change', (e) => {
            dispatch('change', { value: e.target.value });
        });
        group.appendChild(input);
        return group;
    },
    "DataTable": (attributes) => {
        const table = document.createElement('table');
        table.className = 'table table-sm table-hover border mt-2';
        
        const thead = document.createElement('thead');
        thead.innerHTML = `<tr>${(attributes.columns || []).map(c => `<th>${c}</th>`).join('')}</tr>`;
        table.appendChild(thead);

        const tbody = document.createElement('tbody');
        (attributes.rows || []).forEach(row => {
            const tr = document.createElement('tr');
            tr.innerHTML = (attributes.columns || []).map(col => `<td>${row[col] || ''}</td>`).join('');
            tbody.appendChild(tr);
        });
        table.appendChild(tbody);
        return table;
    },
    "Chart": (attributes) => {
        const container = document.createElement('div');
        container.style.height = attributes.height || '300px';
        const canvas = document.createElement('canvas');
        container.appendChild(canvas);

        // Wait for next tick to ensure canvas is in DOM
        setTimeout(() => {
            if (typeof Chart === 'undefined') {
                console.error("Chart.js not loaded");
                return;
            }
            new Chart(canvas, {
                type: attributes.chartType || 'bar',
                data: {
                    labels: attributes.labels || [],
                    datasets: [{
                        label: attributes.label || 'Data',
                        data: attributes.data || [],
                        backgroundColor: 'rgba(54, 162, 235, 0.5)',
                        borderColor: 'rgba(54, 162, 235, 1)',
                        borderWidth: 1
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false
                }
            });
        }, 0);
        return container;
    }
};
