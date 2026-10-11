import {checkUser} from "./util/checkUser.js";
import {single, singleByName, save, saveDefine, define} from "./services/services.js";
import {loadNavBar} from "./components/navbar.js";
import {queryKeys, schemaTypes} from "./models/types.js";
import {getParams} from "./util/searchParamUtil.js";
import {hideOverlay, showOverlay} from "./util/loadingOverlay.js";

const [navBox, headerBox, editorBox, errorBox] = [
    document.querySelector('#nav-box'),
    document.querySelector('#header-box'),
    document.querySelector('#editor-box'),
    document.querySelector('#error-box')
];

let schema;
let editor;

const [type, name, refId, id] = getParams([
    queryKeys.type,
    queryKeys.name,
    queryKeys.refId,
    queryKeys.id,
]);

const lowerType = (type || '').toLowerCase();
if (lowerType === schemaTypes.page) {
    const targetUrl = id ? `page.html?${queryKeys.id}=${encodeURIComponent(id)}` : (refId ? `page.html?${queryKeys.refId}=${encodeURIComponent(refId)}` : 'page.html');
    window.location.replace(targetUrl);
} else if (lowerType === schemaTypes.query) {
    window.location.replace('/graphql');
} else {
    loadNavBar(navBox);
    if (lowerType === schemaTypes.entity) {
        addEntityActionsButtons();
    } else if (lowerType === schemaTypes.menu) {
        addMenuActionsButtons();
    }
    checkUser(() => loadEditor(id, lowerType, name, refId));
}

function addMenuActionsButtons() {
    headerBox.innerHTML = `<button id='saveMenu' class="btn btn-primary">Save Menu</button>`;
    document.getElementById('saveMenu').addEventListener('click', () =>
        submit(schemaTypes.menu, save)
    );
}

function addEntityActionsButtons() {
    headerBox.innerHTML = `
    <button id="saveAndPublish" class="btn btn-primary">Save and Publish</button>
    <button id='saveDefine' class="btn btn-primary">Save Schema</button>
    <button id='editContent' class="btn btn-primary">Edit Content</button>
    <span>
        <input type="checkbox" class="form-check-input" id="showAdvancedActions">
        <label for="showAdvancedActions">Advanced Actions</label>
    </span>
    <span id="advancedEntityActions" hidden>
        <button id='define' class="btn btn-primary">Get Columns Definition from Database</button>
        <button id='saveEntity' class="btn btn-primary">Save Schema Not Update Database</button>
    </span>`;
    
    document.getElementById('saveAndPublish').addEventListener('click', () =>
        submit(schemaTypes.entity, (d)=>saveDefine(d,true))
    );
    document.getElementById('saveDefine').addEventListener('click', () =>
        submit(schemaTypes.entity, (d)=>saveDefine(d,false))
    );

    document.getElementById('editContent').addEventListener('click', function () {
        const val = editor.getValue();
        window.open(`../admin/entities/${val.name}`, '_blank');
    });

    document.getElementById('showAdvancedActions').addEventListener('change', function () {
        document.getElementById('advancedEntityActions').hidden = !this.checked;
    });

    document.getElementById('define').addEventListener('click', getDefine);
    document.getElementById('saveEntity').addEventListener('click', () =>
        submit('entity', save)
    );
}

async function getDefine() {
    const oldValue = editor.getValue();
    const tableName = oldValue.tableName;

    showOverlay();
    const {data, error} = await define(tableName);
    hideOverlay();

    if (data) {
        oldValue.attributes = data.attributes;
        editor.setValue(oldValue);
    } else {
        errorBox.textContent = error;
        errorBox.style.display = 'block';
    }
}

function loadEditor(id, type, name, refId) {
    const lowercaseType = type.toLowerCase();
    editor = new JSONEditor(editorBox, {
        ajax: true,
        schema: {
            $ref: `json/${lowercaseType}.json`,
        },
        compact: true,
        disable_collapse: true,
        disable_array_delete_last_row: true,
        disable_array_delete_all_rows: true,
        disable_properties: true,
        disable_edit_json: false,
        collapsed: true,
        object_layout: "grid",
        show_errors: 'always',
    });

    editor.on('ready', async function () {
        if (id || name || refId) {
            showOverlay();
            const {data: schemaData, error} = id
                ? await single(id)
                : refId
                    ? await single(refId)
                    : await singleByName(name, type);
            hideOverlay();

            if (schemaData) {
                document.title = `${schemaData.name} - ${type} setting - AiGen CMS Schema Builder`;
                const settingData = schemaData.settings?.[type] ||
                    schemaData.settings?.[lowercaseType] ||
                    (schemaData.settings ? schemaData.settings[Object.keys(schemaData.settings).find(k => k.toLowerCase() === lowercaseType)] : null);
                if (refId) {
                    schema = {type, settings: {[type]: settingData}};
                } else {
                    schema = schemaData;
                }
                editor.setValue(settingData);
            } else {
                errorBox.textContent = error;
                errorBox.style.display = 'block';
            }
        } else {
            editor.setValue(null);
        }
    });

    editor.on('change', function () {
        const errors = editor.validate();
        const indicator = document.getElementById('valid_indicator');
        if (errors.length) {
            indicator.style.color = 'red';
            indicator.textContent = 'not valid';
        } else {
            indicator.style.color = 'green';
            indicator.textContent = 'valid';
        }
    });

    return editor;
}

async function submit(type, callback) {
    const errors = editor.validate();
    if (errors.length) return;

    schema = schema || {type};
    schema.settings = {[type]: editor.getValue()};

    showOverlay();
    const {data, error} = await callback(schema);
    hideOverlay();

    if (data) {
        // replace $.toast with basic alert or use a small toast lib alternative if needed
        alert("Submit succeeded!");
        errorBox.textContent = '';
        errorBox.style.display = 'none';
        schema = data;
        editor.setValue(data.settings[type]);
        history.pushState(null, '', `edit.html?${queryKeys.type}=${type}&${queryKeys.id}=${data.id}`);
    } else {
        errorBox.textContent = error;
        errorBox.style.display = 'block';
    }
}
