import { checkUser } from "./util/checkUser.js";
import { loadNavBar } from "./components/navbar.js";
import { toast } from "./utils/toast.js";

checkUser(async () => {
    await loadNavBar('#nav-container');
    loadPlugins();

    document.getElementById('refresh-btn').onclick = loadPlugins;
});

async function loadPlugins() {
    const listContainer = document.getElementById('plugins-list');
    listContainer.innerHTML = '<div class="text-center mt-5"><div class="spinner-border text-primary"></div></div>';

    try {
        const res = await fetch('/api/plugins');
        if (!res.ok) throw new Error("Failed to fetch plugins");
        const plugins = await res.json();
        renderPlugins(plugins);
    } catch (err) {
        toast("Error", err.message, "danger");
        listContainer.innerHTML = '<div class="alert alert-danger">Failed to load plugins.</div>';
    }
}

function renderPlugins(plugins) {
    const listContainer = document.getElementById('plugins-list');
    listContainer.innerHTML = '';

    if (Object.keys(plugins).length === 0) {
        listContainer.innerHTML = '<div class="col-12 text-center mt-5"><p class="text-muted">No plugins found in /plugins directory.</p></div>';
        return;
    }

    Object.values(plugins).forEach(p => {
        const card = document.createElement('div');
        card.className = 'col-md-6 col-lg-4 mb-4';
        
        let statusBadge = '';
        switch(p.status) {
            case 'active': statusBadge = '<span class="badge bg-success badge-status">Active</span>'; break;
            case 'pending': statusBadge = '<span class="badge bg-warning text-dark badge-status">Pending Trust</span>'; break;
            case 'untrusted': statusBadge = '<span class="badge bg-danger badge-status">Untrusted</span>'; break;
            default: statusBadge = `<span class="badge bg-secondary badge-status">${p.status}</span>`;
        }

        const isVerified = p.is_verified ? '<i class="fas fa-check-circle text-primary ms-1" title="Signed & Verified"></i>' : '<i class="fas fa-exclamation-triangle text-warning ms-1" title="Unsigned"></i>';

        card.innerHTML = `
            <div class="card h-100 plugin-card shadow-sm">
                <div class="card-body">
                    <div class="d-flex justify-content-between align-items-start mb-2">
                        <h5 class="card-title mb-0 text-truncate">${p.manifest.name} ${isVerified}</h5>
                        ${statusBadge}
                    </div>
                    <h6 class="card-subtitle mb-2 text-muted small">${p.manifest.id} v${p.manifest.version}</h6>
                    <p class="card-text small text-muted">${p.manifest.description || 'No description provided.'}</p>
                    <div class="mt-3">
                        <button class="btn btn-sm btn-primary mount-btn" data-id="${p.manifest.id}" ${p.status === 'active' ? 'disabled' : ''}>
                            <i class="fas fa-play"></i> Mount
                        </button>
                        <button class="btn btn-sm btn-outline-info settings-btn" data-id="${p.manifest.id}">
                            <i class="fas fa-cog"></i> Config
                        </button>
                    </div>
                </div>
            </div>
        `;
        listContainer.appendChild(card);
    });

    // Attach events
    document.querySelectorAll('.mount-btn').forEach(btn => {
        btn.onclick = async () => {
            const id = btn.getAttribute('data-id');
            try {
                const res = await fetch(`/api/plugins/${id}/mount`, { method: 'POST' });
                if (res.ok) {
                    toast("Success", "Plugin mounted successfully", "success");
                    loadPlugins();
                } else {
                    const msg = await res.text();
                    toast("Error", msg, "danger");
                }
            } catch (err) {
                toast("Error", err.message, "danger");
            }
        };
    });

    document.querySelectorAll('.settings-btn').forEach(btn => {
        btn.onclick = () => {
            const id = btn.getAttribute('data-id');
            showConfigModal(plugins[id]);
        };
    });
}

function showConfigModal(plugin) {
    const modalBody = document.getElementById('modal-body');
    const manifest = plugin.manifest;

    let permissionsHtml = '<h6>Permissions</h6><ul class="list-group mb-3">';
    if (manifest.permissions && manifest.permissions.length > 0) {
        manifest.permissions.forEach(perm => {
            permissionsHtml += `
                <li class="list-group-item d-flex justify-content-between align-items-center">
                    <div>
                        <span class="badge bg-light text-dark me-2">${perm.type}</span>
                        <code>${perm.value}</code>
                    </div>
                    <button class="btn btn-sm btn-outline-success authorize-btn" data-id="${manifest.id}" data-type="${perm.type}" data-value="${perm.value}">
                        Authorize
                    </button>
                </li>
            `;
        });
    } else {
        permissionsHtml += '<li class="list-group-item text-muted small">No permissions requested.</li>';
    }
    permissionsHtml += '</ul>';

    let secretsHtml = '<h6>Vault Secrets</h6><div class="mb-3">';
    if (manifest.env_vars && manifest.env_vars.length > 0) {
        manifest.env_vars.forEach(key => {
            secretsHtml += `
                <div class="input-group mb-2">
                    <span class="input-group-text small">${key}</span>
                    <input type="password" class="form-control form-control-sm secret-input" data-plugin="${manifest.id}" data-key="${key}" placeholder="Enter value">
                    <button class="btn btn-sm btn-primary save-secret-btn">Save</button>
                </div>
            `;
        });
    } else {
        secretsHtml += '<p class="text-muted small">No environment variables requested.</p>';
    }
    secretsHtml += '</div>';

    modalBody.innerHTML = `
        <div class="mb-3">
            <strong>Author:</strong> ${manifest.author || 'Unknown'}<br>
            <strong>Signer:</strong> ${plugin.signer || 'N/A'}
        </div>
        <hr>
        ${permissionsHtml}
        <hr>
        ${secretsHtml}
    `;

    // Attach events
    modalBody.querySelectorAll('.authorize-btn').forEach(btn => {
        btn.onclick = async () => {
            const id = btn.getAttribute('data-id');
            const type = btn.getAttribute('data-type');
            const value = btn.getAttribute('data-value');
            try {
                const res = await fetch(`/api/plugins/${id}/authorize`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ type, value })
                });
                if (res.ok) {
                    toast("Success", "Permission authorized", "success");
                    btn.classList.replace('btn-outline-success', 'btn-success');
                    btn.innerText = 'Authorized';
                    btn.disabled = true;
                }
            } catch (err) {
                toast("Error", err.message, "danger");
            }
        };
    });

    modalBody.querySelectorAll('.save-secret-btn').forEach(btn => {
        btn.onclick = async () => {
            const input = btn.previousElementSibling;
            const id = input.getAttribute('data-plugin');
            const key = input.getAttribute('data-key');
            const value = input.value;

            if (!value) return toast("Error", "Value cannot be empty", "warning");

            try {
                const res = await fetch(`/api/plugins/${id}/secrets`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ key, value })
                });
                if (res.ok) {
                    toast("Success", `Secret ${key} saved`, "success");
                    input.value = '';
                }
            } catch (err) {
                toast("Error", err.message, "danger");
            }
        };
    });

    const modal = new bootstrap.Modal(document.getElementById('permissionModal'));
    modal.show();
}
