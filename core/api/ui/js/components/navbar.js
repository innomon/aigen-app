import { logout } from "../services/services.js";
import { fetchUser, getActiveRole, setActiveRoleId } from "../utils/user.js";

export async function loadNavBar(container) {
    const user = await fetchUser();
    const activeRole = getActiveRole();
    let menuItemsHtml = '';

    let hasCustomMenu = false;
    if (activeRole && activeRole.menuId) {
        try {
            const res = await fetch(`/api/schemas/${activeRole.menuId}`);
            if (res.ok) {
                const schema = await res.json();
                if (schema && schema.menu && schema.menu.menuItems) {
                    hasCustomMenu = true;
                    menuItemsHtml = schema.menu.menuItems.map(item => 
                        `<a class="nav-item nav-link border-item" href="${item.url}">${item.icon ? `<i class="${item.icon}"></i> ` : ''}${item.label}</a>`
                    ).join('');
                }
            }
        } catch (e) {
            console.error("Failed to load custom menu", e);
        }
    }

    if (!hasCustomMenu) {
        menuItemsHtml = `
            <a class="nav-item nav-link border-item" href="./list.html">All Schemas</a>
            <a class="nav-item nav-link border-item" href="./list.html?type=entity">Entities</a>
            <a class="nav-item nav-link border-item" href="./list.html?type=query">Queries</a>
            <a class="nav-item nav-link border-item" href="./list.html?type=page">Pages</a>
            <a class="nav-item nav-link border-item" href="./edit.html?type=menu&name=top-menu-bar">MenuItems</a>
            <a class="nav-item nav-link border-item" href="./entity_list.html?name=User">Users</a>
            <a class="nav-item nav-link border-item" href="./entity_list.html?name=Role">Roles</a>
            <a class="nav-item nav-link border-item" href="../admin">Admin Panel</a>`;
    }

    let roleSwitcherHtml = '';
    if (user && user.rolesDetails && user.rolesDetails.length > 1) {
        roleSwitcherHtml = `
            <select id="role-switcher" class="form-select w-auto d-inline-block ms-3">
                ${user.rolesDetails.map(r => `<option value="${r.id}" ${activeRole && activeRole.id === r.id ? 'selected' : ''}>${r.name}</option>`).join('')}
            </select>
        `;
    }

    const html = `
    <nav class="navbar navbar-expand-lg navbar-light bg-light">
        <a class="navbar-brand" href="/">
            <img alt="logo" src="img/aigen-cms.svg" height="50" class="mr-2">
        </a>
        <div class="navbar-nav w-100 d-flex align-items-center">
            ${menuItemsHtml}
            <a id="nav-item-exit" class="nav-item nav-link border-item ms-auto" href="#">Exit</a>
            ${roleSwitcherHtml}
        </div>
    </nav>`;

    if (typeof container === "string") {
        document.querySelector(container).innerHTML = html;
    } else if (container instanceof HTMLElement) {
        container.innerHTML = html;
    }

    const exitLink = document.getElementById("nav-item-exit");
    if (exitLink) {
        exitLink.addEventListener("click", function (e) {
            e.preventDefault();
            logout().then(() => {
                window.location.href = "/";
            });
        });
    }

    const roleSwitcher = document.getElementById("role-switcher");
    if (roleSwitcher) {
        roleSwitcher.addEventListener("change", function (e) {
            setActiveRoleId(e.target.value);
        });
    }
}
