# Role-Based Page Lifecycle Management

## 1. Overview
In the FormCMS Go system, user interfaces—specifically dashboards and navigation menus—are dynamically linked to User Roles. This document outlines the lifecycle management of these role-based pages, covering the end-to-end process from initial design and development to pre-deployment preparation and post-deployment runtime management.

This dynamic approach allows FormCMS to function as a fully headless CMS while simultaneously providing tailored, built-in administrative or user portal experiences that evolve without requiring backend code changes.

---

## 2. Design Phase

The design phase focuses on understanding the requirements of a specific user persona and mapping those requirements to the FormCMS capabilities.

### 2.1. Persona Definition
* **Identify the Role:** Clearly define the target role (e.g., `Sales Manager`, `Content Editor`, `Customer Support`).
* **Determine Access:** What entities and actions (Read, Write, Create, Delete) does this role require? This dictates the Document Permissions (`DocPerms`) needed.

### 2.2. UI/UX Planning
* **Dashboard Layout:** Plan the visual layout of the dashboard. What key metrics, charts, or shortcut buttons does this role need immediately upon logging in?
* **Navigation Architecture:** Design the navigation menu. Which pages or entity lists should be readily accessible in the top-level or sidebar navigation?

---

## 3. Development Phase

The development phase involves physically creating the assets within a local or development instance of FormCMS.

### 3.1. Asset Creation via Admin Panel
1. **Create the Menu:** 
   * Navigate to the Admin Panel -> MenuItems.
   * Create a new `Menu` entity (e.g., `sales_manager_menu`).
   * Add the required navigation links (e.g., shortcuts to Leads, Deals, Reports).
2. **Design the Dashboard Page:**
   * Navigate to the Admin Panel -> Pages.
   * Create a new `Page` (e.g., `sales_dashboard`).
   * Use the integrated **GrapesJS** visual page builder to construct the layout, dragging in necessary components, styling the HTML/CSS, and linking dynamic data widgets if applicable.
   * Save the Page and note its generated ID.
3. **Configure the Role:**
   * Navigate to Admin Panel -> Roles.
   * Create or edit the target Role.
   * Set the `dashboard_page_id` to the ID of the newly created Page.
   * Set the `menu_id` to the ID of the newly created Menu.

### 3.2. Local Testing
* Create a test User and assign them the new Role.
* Set the user's `default_role_id` to this Role.
* Log in as the test User to verify:
  * The correct dashboard page loads automatically.
  * The custom navigation menu is displayed.
  * The role-switcher dropdown functions correctly if the user is assigned multiple roles.

---

## 4. Pre-Deployment Phase

To ensure environments are reproducible, the dynamic assets created in the Development phase must be captured as code.

### 4.1. Exporting Configuration as Code
FormCMS allows entities to be exported to JSON formats.
* **Export Pages & Menus:** Export the raw data of the `Page` and `Menu` entities.
* **Export Roles:** Export the `Role` configurations, ensuring the references (`dashboard_page_id` and `menu_id`) are preserved.

### 4.2. Seeding Data Files
* Place the exported JSON objects into the appropriate initialization files (e.g., `apps/rbac/data/test_data.json` or specific deployment seed files).
* *Example Data Structure:*
  ```json
  [
    {
      "Entity": "Menu",
      "Ref": "menu_sales",
      "Data": { "name": "Sales Menu", "menuItems": [...] }
    },
    {
      "Entity": "Page",
      "Ref": "page_sales_dash",
      "Data": { "Html": "...", "Css": "..." }
    },
    {
      "Entity": "Role",
      "Ref": "role_sales",
      "Data": { 
        "name": "Sales Manager", 
        "menu_id": "menu_sales", 
        "dashboard_page_id": "page_sales_dash" 
      }
    }
  ]
  ```

### 4.3. Version Control
* Commit these JSON data files into the Git repository. This guarantees that any new environment spun up will have the exact dashboard and menu configurations desired.

---

## 5. Deployment Phase

During the deployment of the FormCMS Go application to staging or production environments, the system automatically provisions the configuration.

### 5.1. Automated Ingestion
* Upon system startup (or triggered via an import CLI command/API), FormCMS reads the JSON seed files.
* The system performs "Upsert" operations. If the `Role`, `Page`, or `Menu` does not exist, it is created. If it does exist, it is updated to match the definition in the codebase.
* **Referential Integrity:** The import process ensures that `Ref` tags are resolved correctly to physical database IDs, ensuring the `Role` accurately points to the newly inserted `Page` and `Menu`.

---

## 6. Post-Deployment Phase

Once live in production, the dynamic nature of FormCMS allows for ongoing management without development cycles.

### 6.1. Runtime Modifications
* System Administrators can log into the Production Admin Panel to make live adjustments.
* **Tweaking Dashboards:** An admin can open the `Page` in the GrapesJS editor, alter the layout or text, and click Save. The changes are immediately reflected for all users holding that Role.
* **Updating Menus:** Adding a new link to a role's Menu is done directly through the UI, instantly updating the navigation for affected users.

### 6.2. Fallback and Error Handling
* **Missing Assets:** If a `dashboard_page_id` is accidentally deleted or corrupted in production, the system's fallback mechanism activates. When the user logs in, the backend gracefully handles the missing page and serves the default System Dashboard, ensuring the user is never locked out of the application.

### 6.3. Continuous Iteration
* If significant changes are made directly in Production (e.g., a completely redesigned dashboard), these changes should be periodically exported from the Production database back into the repository's JSON seed files to prevent configuration drift and ensure disaster recovery readiness.