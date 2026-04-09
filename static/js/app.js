// Flow Manager - Frontend Logic

const csrfToken = document.querySelector('meta[name="csrf-token"]').getAttribute('content');

async function generateMarkdownContent() {
    const form = document.getElementById('flow-form');
    const formData = new FormData(form);
    formData.set('action', 'markdown');

    try {
        const res = await fetch('/submit', {
            method: 'POST',
            headers: { 'X-CSRF-TOKEN': csrfToken },
            body: formData
        });
        if (res.ok) {
            const data = await res.json();
            document.getElementById('markdownContent').value = data.markdown;
            const modal = new bootstrap.Modal(document.getElementById('markdownModal'));
            modal.show();
        } else {
            alert('Erreur lors de la génération du Markdown');
        }
    } catch (err) {
        console.error(err);
        alert('Erreur réseau');
    }
}

function copyMarkdown() {
    const textarea = document.getElementById('markdownContent');
    textarea.select();
    document.execCommand('copy');
    alert('Markdown copié !');
}

function addRow() {
    const tbody = document.getElementById('form-body');
    if (!tbody) return;
    const idx = rowIndex++;
    const row = document.createElement('tr');
    row.id = `row-${idx}`; row.className = 'form-row-inputs';
    let scopeOptions = `<option value="">-- Machine isolée --</option><option value="EXTERNAL">-- Externe (Hostname) --</option>`;
    availableVlans.forEach(v => { scopeOptions += `<option value="${v.subnet}">${v.name} (${v.subnet})</option>`; });
    row.innerHTML = `
        <td><button type="button" class="btn btn-danger btn-sm" onclick="this.closest('tr').remove()">X</button></td>
        <td><select class="form-select form-select-sm mb-1" onchange="selectScope(this, ${idx}, 'source')">${scopeOptions}</select><input type="text" name="flows[${idx}].source_vlan" class="form-control" id="source_vlan_${idx}" readonly placeholder="VLAN auto"></td>
        <td><input type="text" name="flows[${idx}].source_ip" class="form-control" id="source_ip_${idx}" placeholder="IP ou Subnet" list="ciSuggestions" oninput="updateSuggestions(this.value)"></td>
        <td><input type="text" name="flows[${idx}].source_hostname" class="form-control" id="source_hostname_${idx}" placeholder="Hostname" list="ciSuggestions" oninput="updateSuggestions(this.value)"></td>
        <td><select class="form-select form-select-sm mb-1" onchange="selectScope(this, ${idx}, 'target')">${scopeOptions}</select><input type="text" name="flows[${idx}].target_vlan" class="form-control" id="target_vlan_${idx}" readonly placeholder="VLAN auto"></td>
        <td><input type="text" name="flows[${idx}].target_ip" class="form-control" id="target_ip_${idx}" placeholder="IP ou Subnet" list="ciSuggestions" oninput="updateSuggestions(this.value)"></td>
        <td><input type="text" name="flows[${idx}].target_hostname" class="form-control" id="target_hostname_${idx}" placeholder="Hostname" list="ciSuggestions" oninput="updateSuggestions(this.value)"></td>
        <td><select name="flows[${idx}].protocol" class="form-select"><option value="TCP">TCP</option><option value="UDP">UDP</option><option value="BOTH">TCP & UDP</option></select></td>
        <td><input type="text" name="flows[${idx}].ports" class="form-control" placeholder="80, 443"></td>
        <td><input type="date" name="flows[${idx}].time_limit" class="form-control"></td>
        <td><input type="text" name="flows[${idx}].comment" class="form-control"></td>
    `;
    tbody.appendChild(row);
    ['source', 'target'].forEach(t => { 
        [`${t}_hostname_${idx}`, `${t}_ip_${idx}`].forEach(id => { 
            const el = document.getElementById(id);
            if (el) el.addEventListener('change', e => lookupCI(e.target.value, idx, t)); 
        }); 
    });
}

function selectScope(selectEl, idx, type) {
    const val = selectEl.value; const ipInput = document.getElementById(`${type}_ip_${idx}`);
    const hostnameInput = document.getElementById(`${type}_hostname_${idx}`); const vlanNameInput = document.getElementById(`${type}_vlan_${idx}`);
    if (val === "EXTERNAL") {
        ipInput.value = ""; ipInput.disabled = true; hostnameInput.disabled = false; hostnameInput.placeholder = "Hostname (ex: machine-externe)"; vlanNameInput.value = "EXTERNE";
    } else if (val) {
        ipInput.value = val; ipInput.disabled = false; hostnameInput.value = ""; hostnameInput.disabled = true;
        const vlan = availableVlans.find(v => v.subnet === val); if (vlan) vlanNameInput.value = vlan.name;
    } else {
        ipInput.disabled = false; hostnameInput.disabled = false; hostnameInput.placeholder = "Hostname"; ipInput.placeholder = "IP ou Subnet"; vlanNameInput.value = "";
    }
}

async function updateSuggestions(query) {
    if (query.length < 2) return;
    try {
        const res = await fetch(`/ci/suggest?query=${encodeURIComponent(query)}`);
        if (res.ok) {
            const cis = await res.json(); const dl = document.getElementById('ciSuggestions'); dl.innerHTML = '';
            cis.forEach(ci => {
                const oH = document.createElement('option'); oH.value = ci.hostname; dl.appendChild(oH);
                const oI = document.createElement('option'); oI.value = ci.ip; dl.appendChild(oI);
            });
        }
    } catch (err) { console.error(err); }
}

async function lookupCI(q, idx, type) {
    if (!q) return;
    const h = document.getElementById(`${type}_hostname_${idx}`), i = document.getElementById(`${type}_ip_${idx}`), v = document.getElementById(`${type}_vlan_${idx}`);
    try {
        let res = await fetch(`/ci/lookup?query=${encodeURIComponent(q)}`);
        if (res.ok) {
            let ci = await res.json(); if (ci.hostname) h.value = ci.hostname; if (ci.ip) i.value = ci.ip;
            v.value = ci.vlan?.vlan || "";
        } else if (q.match(/^(\d{1,3}\.){3}\d{1,3}$/)) {
            let vres = await fetch(`/vlan/lookup?ip=${encodeURIComponent(q)}`);
            if (vres.ok) { let data = await vres.json(); if (data.vlan?.vlan) v.value = data.vlan.vlan; }
        }
    } catch (err) { console.error(err); }
}

function prepareVlanModal(v) {
    document.getElementById('vlanId').value = v?.ID || ''; document.getElementById('vlanSubnet').value = v?.Subnet || '';
    document.getElementById('vlanName').value = v?.VLAN || ''; document.getElementById('vlanGateway').value = v?.Gateway || '';
    document.getElementById('vlanDNSServers').value = v?.DNSServers || '';
}

async function saveVlan() {
    const id = document.getElementById('vlanId').value;
    const body = { subnet: document.getElementById('vlanSubnet').value, vlan: document.getElementById('vlanName').value, gateway: document.getElementById('vlanGateway').value, dns_servers: document.getElementById('vlanDNSServers').value };
    const res = await fetch(id ? `/vlan/${id}` : '/vlan', { 
        method: id ? 'PUT' : 'POST', 
        headers: {'Content-Type': 'application/json', 'X-CSRF-TOKEN': csrfToken}, 
        body: JSON.stringify(body) 
    });
    if (res.ok) window.location.href = '/?tab=vlan'; else alert('Erreur');
}

async function deleteVlan(id) { 
    if (confirm('Supprimer ce VLAN ?')) { 
        await fetch(`/vlan/${id}`, {method: 'DELETE', headers: {'X-CSRF-TOKEN': csrfToken}}); 
        window.location.href = '/?tab=vlan'; 
    } 
}

async function uploadVlanCSV() {
    const input = document.getElementById('csvFileInput'); if (input.files.length === 0) return;
    const formData = new FormData(); 
    formData.append('file', input.files[0]);
    formData.append('_csrf', csrfToken);
    const res = await fetch('/vlan/import', {method: 'POST', body: formData});
    if (res.ok) { const data = await res.json(); alert(`Import réussi : ${data.created} créés, ${data.updated} mis à jour.`); window.location.href = '/?tab=vlan'; } else alert('Erreur');
}

function prepareCiModal(ci) {
    document.getElementById('ciId').value = ci?.ID || ''; document.getElementById('ciHostname').value = ci?.Hostname || '';
    document.getElementById('ciIP').value = ci?.IP || ''; document.getElementById('ciDescription').value = ci?.Description || '';
}

async function saveCi() {
    const id = document.getElementById('ciId').value;
    const body = { hostname: document.getElementById('ciHostname').value, ip: document.getElementById('ciIP').value, description: document.getElementById('ciDescription').value };
    const res = await fetch(id ? `/ci/${id}` : '/ci', { 
        method: id ? 'PUT' : 'POST', 
        headers: {'Content-Type': 'application/json', 'X-CSRF-TOKEN': csrfToken}, 
        body: JSON.stringify(body) 
    });
    if (res.ok) window.location.href = '/?tab=ci'; else { let e = await res.json(); alert(e.error); }
}

async function deleteCi(id) { 
    if (confirm('Supprimer ce CI ?')) { 
        await fetch(`/ci/${id}`, {method: 'DELETE', headers: {'X-CSRF-TOKEN': csrfToken}}); 
        window.location.href = '/?tab=ci'; 
    } 
}

async function uploadCiCSV() {
    const input = document.getElementById('ciCsvFileInput'); if (input.files.length === 0) return;
    const formData = new FormData(); 
    formData.append('file', input.files[0]);
    formData.append('_csrf', csrfToken);
    const res = await fetch('/ci/import', {method: 'POST', body: formData});
    if (res.ok) { const data = await res.json(); alert(`Import réussi : ${data.created} créés, ${data.updated} mis à jour.`); window.location.href = '/?tab=ci'; } else alert('Erreur');
}

function prepareFlowEditModal(f) {
    document.getElementById('editFlowId').value = f.ID; document.getElementById('editRuleNumber').value = f.RuleNumber;
    document.getElementById('editStatus').value = f.Status; document.getElementById('editComment').value = f.Comment;
}

async function saveFlowEdit() {
    const id = document.getElementById('editFlowId').value;
    const res = await fetch(`/flow/${id}`, { 
        method: 'PUT', 
        headers: {'Content-Type': 'application/json', 'X-CSRF-TOKEN': csrfToken}, 
        body: JSON.stringify({ rule_number: document.getElementById('editRuleNumber').value, status: document.getElementById('editStatus').value, comment: document.getElementById('editComment').value }) 
    });
    if (res.ok) window.location.reload(); else alert('Erreur');
}

function prepareUserModal(u) {
    document.getElementById('userId').value = u?.ID || ''; document.getElementById('userUsername').value = u?.Username || '';
    document.getElementById('userPassword').value = ''; document.getElementById('userRole').value = u?.Role || 'viewer';
    document.getElementById('userModalLabel').innerText = u ? 'Modifier l\'utilisateur' : 'Ajouter un utilisateur';
}

async function saveUser() {
    const id = document.getElementById('userId').value;
    const body = { username: document.getElementById('userUsername').value, password: document.getElementById('userPassword').value, role: document.getElementById('userRole').value };
    const res = await fetch(id ? `/users/${id}` : '/users', { 
        method: id ? 'PUT' : 'POST', 
        headers: {'Content-Type': 'application/json', 'X-CSRF-TOKEN': csrfToken}, 
        body: JSON.stringify(body) 
    });
    if (res.ok) window.location.href = '/?tab=admin'; else alert('Erreur lors de la sauvegarde');
}

async function deleteUser(id) { 
    if (confirm('Supprimer cet utilisateur ?')) { 
        await fetch(`/users/${id}`, {method: 'DELETE', headers: {'X-CSRF-TOKEN': csrfToken}}); 
        window.location.href = '/?tab=admin'; 
    } 
}

document.addEventListener('DOMContentLoaded', () => { 
    if(document.getElementById('home-tab')?.classList.contains('active')) addRow(); 
});
