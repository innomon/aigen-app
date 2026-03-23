import { checkUser } from "./util/checkUser.js";
import { loadNavBar } from "./components/navbar.js";
import { fetchUser, getActiveRole } from "./utils/user.js";

checkUser(async () => {
    await loadNavBar('#nav-container');
    const user = await fetchUser();
    const activeRole = getActiveRole();
    const dashboardContent = document.getElementById('dashboard-content');

    let loadedCustomDashboard = false;

    if (activeRole && activeRole.dashboardPageId) {
        try {
            const res = await fetch(`/api/pages/${activeRole.dashboardPageId}`);
            if (res.ok) {
                const pageData = await res.json();
                if (pageData && pageData.Html) {
                    // Render HTML and CSS
                    let html = pageData.Html || '';
                    if (pageData.Css) {
                        html = `<style>${pageData.Css}</style>` + html;
                    }
                    dashboardContent.innerHTML = html;
                    loadedCustomDashboard = true;
                }
            }
        } catch (e) {
            console.error("Failed to load dashboard page", e);
        }
    }

    if (!loadedCustomDashboard) {
        // Fallback default dashboard
        dashboardContent.innerHTML = `
            <div class="row mt-4">
                <div class="col-12 text-center">
                    <h2>Welcome to AiGen CMS</h2>
                    <p class="text-muted">You are logged in as <strong>${user.email}</strong> with role <strong>${activeRole ? activeRole.name : 'Unknown'}</strong>.</p>
                </div>
            </div>
            <div class="row mt-4 justify-content-center">
                <div class="col-md-4 mb-3">
                    <div class="card text-center shadow-sm h-100">
                        <div class="card-body">
                            <i class="fas fa-database fa-3x mb-3 text-primary"></i>
                            <h5 class="card-title">Manage Data</h5>
                            <p class="card-text">View and manage system entities.</p>
                            <a href="./list.html?type=entity" class="btn btn-primary">Go to Entities</a>
                        </div>
                    </div>
                </div>
                <div class="col-md-4 mb-3">
                    <div class="card text-center shadow-sm h-100">
                        <div class="card-body">
                            <i class="fas fa-users fa-3x mb-3 text-success"></i>
                            <h5 class="card-title">User Management</h5>
                            <p class="card-text">Manage users and roles.</p>
                            <a href="./entity_list.html?name=User" class="btn btn-success">Go to Users</a>
                        </div>
                    </div>
                </div>
            </div>
        `;
    }
});