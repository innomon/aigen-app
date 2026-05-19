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
    },
    "UcpProductCard": (attributes, children, dispatch) => {
        const card = document.createElement('div');
        card.className = 'card h-100 ucp-product-card shadow-sm';
        
        const img = document.createElement('img');
        img.src = attributes.image || 'https://via.placeholder.com/150';
        img.className = 'card-img-top p-3';
        img.style.maxHeight = '200px';
        img.style.objectFit = 'contain';
        card.appendChild(img);

        const body = document.createElement('div');
        body.className = 'card-body d-flex flex-column';
        
        const title = document.createElement('h5');
        title.className = 'card-title';
        title.textContent = attributes.name;
        body.appendChild(title);

        const desc = document.createElement('p');
        desc.className = 'card-text small text-muted flex-grow-1';
        desc.textContent = attributes.description;
        body.appendChild(desc);

        const priceRow = document.createElement('div');
        priceRow.className = 'd-flex justify-content-between align-items-center mt-3';
        
        const price = document.createElement('span');
        price.className = 'h5 mb-0 text-primary';
        price.textContent = `${attributes.currency} ${attributes.price}`;
        priceRow.appendChild(price);

        const buyBtn = document.createElement('button');
        buyBtn.className = 'btn btn-sm btn-success';
        buyBtn.innerHTML = '<i class="bi bi-cart-plus"></i> Buy Now';
        buyBtn.onclick = () => dispatch('buy', { product_id: attributes.product_id });
        priceRow.appendChild(buyBtn);

        body.appendChild(priceRow);
        card.appendChild(body);
        return card;
    },
    "UcpCheckoutSummary": (attributes, children, dispatch) => {
        const div = document.createElement('div');
        div.className = 'list-group shadow-sm';
        
        const header = document.createElement('div');
        header.className = 'list-group-item active d-flex justify-content-between align-items-center';
        header.innerHTML = `<span><i class="bi bi-cart-check"></i> Checkout Summary</span> <span class="badge bg-light text-dark">${attributes.status}</span>`;
        div.appendChild(header);

        (attributes.items || []).forEach(item => {
            const row = document.createElement('div');
            row.className = 'list-group-item d-flex justify-content-between align-items-center';
            row.innerHTML = `<div><div class="fw-bold">${item.name}</div><small class="text-muted">Qty: ${item.quantity}</small></div> <span>${attributes.currency} ${item.price}</span>`;
            div.appendChild(row);
        });

        const totalRow = document.createElement('div');
        totalRow.className = 'list-group-item list-group-item-secondary d-flex justify-content-between align-items-center fw-bold';
        totalRow.innerHTML = `<span>Total</span> <span>${attributes.currency} ${attributes.total}</span>`;
        div.appendChild(totalRow);

        if (attributes.status === 'active') {
            const footer = document.createElement('div');
            footer.className = 'list-group-item';
            const payBtn = document.createElement('button');
            payBtn.className = 'btn btn-primary w-100';
            payBtn.textContent = 'Authorize Payment (AP2)';
            payBtn.onclick = () => dispatch('authorize');
            footer.appendChild(payBtn);
            div.appendChild(footer);
        }

        return div;
    },
    "Ap2MandateBanner": (attributes) => {
        const div = document.createElement('div');
        const isValid = attributes.verified === true;
        div.className = `alert alert-${isValid ? 'success' : 'warning'} d-flex align-items-center shadow-sm`;
        div.role = 'alert';
        
        const icon = document.createElement('i');
        icon.className = `bi bi-${isValid ? 'shield-check' : 'shield-exclamation'} me-3 h4 mb-0`;
        div.appendChild(icon);

        const content = document.createElement('div');
        content.innerHTML = `<strong>AP2 Mandate ${isValid ? 'Verified' : 'Unverified'}</strong><br><small class="text-break">Key: ${attributes.public_key}</small>`;
        div.appendChild(content);

        return div;
    }
};
