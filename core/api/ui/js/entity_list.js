import { checkUser } from "./util/checkUser.js";
import { loadNavBar } from "./components/navbar.js";
import { getJson } from "./services/util.js";
import { getParams } from "./util/searchParamUtil.js";
import { showOverlay, hideOverlay } from "./util/loadingOverlay.js";

checkUser(async () => {
    const navBox = document.querySelector('#nav-box');
    const tableBox = document.querySelector('#table-box');
    const errorBox = document.querySelector('#error-box');
    const entityTitle = document.querySelector('#entity-title');
    const actionsBox = document.querySelector('#actions-box');

    const [name] = getParams(['name']);
    if (!name) {
        errorBox.textContent = "Entity name is required";
        errorBox.style.display = 'block';
        return;
    }

    entityTitle.textContent = `${name} List`;
    loadNavBar(navBox);
    actionsBox.innerHTML = `<a href="entity_edit.html?name=${name}" class="btn btn-primary">Add ${name}</a>`;

    showOverlay();
    // Fetch schema to know which columns to show
    const schemaRes = await getJson(`/schemas/name/${name}?type=entity`);
    if (schemaRes.error) {
        hideOverlay();
        errorBox.textContent = schemaRes.error;
        errorBox.style.display = 'block';
        return;
    }

    const entitySchema = schemaRes.settings.entity;
    const listCols = entitySchema.attributes.filter(a => a.inList);

    // Fetch entity data
    const dataRes = await getJson(`/entities/${name}`);
    hideOverlay();

    if (dataRes.error) {
        errorBox.textContent = dataRes.error;
        errorBox.style.display = 'block';
        return;
    }

    const records = dataRes.items;
    
    let tableHtml = `
        <table class="table table-striped table-hover">
            <thead>
                <tr>
                    <th>ID</th>
                    ${listCols.map(col => `<th>${col.header}</th>`).join('')}
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                ${records.map(rec => `
                    <tr>
                        <td>${rec[entitySchema.primaryKey]}</td>
                        ${listCols.map(col => `<td>${rec[col.field] ?? ''}</td>`).join('')}
                        <td>
                            <a href="entity_edit.html?name=${name}&id=${rec[entitySchema.primaryKey]}" class="btn btn-sm btn-outline-primary">Edit</a>
                            <button class="btn btn-sm btn-outline-danger delete-btn" data-id="${rec[entitySchema.primaryKey]}">Delete</button>
                        </td>
                    </tr>
                `).join('')}
            </tbody>
        </table>
    `;

    tableBox.innerHTML = tableHtml;

    document.querySelectorAll('.delete-btn').forEach(btn => {
        btn.addEventListener('click', async (e) => {
            const id = e.target.getAttribute('data-id');
            if (confirm(`Are you sure you want to delete this ${name}?`)) {
                showOverlay();
                const res = await fetch(`/api/entities/${name}/${id}`, { method: 'DELETE', credentials: 'include' });
                hideOverlay();
                if (res.ok) {
                    window.location.reload();
                } else {
                    const text = await res.text();
                    errorBox.textContent = text || "Delete failed";
                    errorBox.style.display = 'block';
                }
            }
        });
    });
});
