import { checkUser } from "./util/checkUser.js";
import { loadNavBar } from "./components/navbar.js";
import { getJson, postData } from "./services/util.js";
import { getParams } from "./util/searchParamUtil.js";
import { showOverlay, hideOverlay } from "./util/loadingOverlay.js";

checkUser(async () => {
    const navBox = document.querySelector('#nav-box');
    const editorBox = document.querySelector('#editor-box');
    const errorBox = document.querySelector('#error-box');
    const entityTitle = document.querySelector('#entity-title');
    const saveBtn = document.querySelector('#save-btn');

    const [name, id] = getParams(['name', 'id']);
    if (!name) {
        errorBox.textContent = "Entity name is required";
        errorBox.style.display = 'block';
        return;
    }

    loadNavBar(navBox);
    entityTitle.textContent = id ? `Edit ${name} #${id}` : `Add ${name}`;

    showOverlay();
    const schemaRes = await getJson(`/schemas/name/${name}?type=entity`);
    if (schemaRes.error) {
        hideOverlay();
        errorBox.textContent = schemaRes.error;
        errorBox.style.display = 'block';
        return;
    }

    const entitySchema = schemaRes.settings.entity;
    
    // Build JSON Schema from entity attributes
    const properties = {};
    const required = [];
    
    entitySchema.attributes.forEach(attr => {
        let type = 'string';
        let format = '';
        
        switch(attr.dataType) {
            case 'Int': type = 'integer'; break;
            case 'Float': type = 'number'; break;
            case 'Boolean': type = 'boolean'; break;
            case 'Datetime': type = 'string'; format = 'datetime'; break;
        }
        
        if (attr.displayType === 'Password') {
            format = 'password';
        }

        properties[attr.field] = {
            type: type,
            title: attr.header || attr.field,
        };
        
        if (format) {
            properties[attr.field].format = format;
        }
        
        if (attr.required) {
            required.push(attr.field);
        }
    });

    const jsonSchema = {
        type: 'object',
        properties: properties,
        required: required
    };

    const editor = new JSONEditor(editorBox, {
        schema: jsonSchema,
        theme: 'bootstrap5',
        disable_edit_json: true,
        disable_properties: true,
        disable_collapse: true
    });

    if (id) {
        const dataRes = await getJson(`/entities/${name}/${id}`);
        if (dataRes.error) {
            errorBox.textContent = dataRes.error;
            errorBox.style.display = 'block';
        } else {
            editor.setValue(dataRes);
        }
    }
    hideOverlay();

    saveBtn.addEventListener('click', async () => {
        const errors = editor.validate();
        if (errors.length) {
            errorBox.textContent = "Please fix validation errors";
            errorBox.style.display = 'block';
            return;
        }

        const data = editor.getValue();
        if (id) {
            data[entitySchema.primaryKey] = id;
        }

        showOverlay();
        const url = id ? `/entities/${name}` : `/entities/${name}`;
        const method = id ? 'PUT' : 'POST';
        
        const response = await fetch(`/api${url}`, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data),
            credentials: 'include'
        });
        
        hideOverlay();
        if (response.ok) {
            window.location.href = `entity_list.html?name=${name}`;
        } else {
            const text = await response.text();
            errorBox.textContent = text || "Save failed";
            errorBox.style.display = 'block';
        }
    });
});
