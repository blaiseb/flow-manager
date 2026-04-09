// Flow Manager - Frontend Logic

const csrfToken = document.querySelector('meta[name="csrf-token"]').getAttribute('content');
let standardFlows = {};

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

async function fetchStandardFlows() {
    try {
        const res = await fetch('/standard-flows');
        if (res.ok) {
            const data = await res.json();
            standardFlows = { 'custom': { protocol: 'TCP', ports: '' } };
            data.forEach(f => {
                standardFlows[f.name.toLowerCase()] = { protocol: f.protocol, ports: f.ports };
            });
            updateTypeSelects();
        }
    } catch (err) { console.error("Failed to fetch standard flows", err); }
}

function updateTypeSelects() {
    const selects = document.querySelectorAll('select[id^="flow_type_"]');
    selects.forEach(select => {
        const currentVal = select.value;
        select.innerHTML = '<option value="custom">Manuel</option>';
        Object.keys(standardFlows).forEach(key => {
            if (key !== 'custom') {
                const opt = document.createElement('option');
                opt.value = key;
                opt.textContent = key.toUpperCase();
                select.appendChild(opt);
            }
        });
        select.value = currentVal;
    });
}

function addRow() {
    const tbody = document.getElementById('form-body');
    if (!tbody) return;
    const idx = rowIndex++;
    const row = document.createElement('tr');
    row.id = `row-${idx}`; row.className = 'form-row-inputs';
    let scopeOptions = `<option value="">-- Machine isolée --</option><option value="EXTERNAL">-- Externe (Hostname) --</option>`;
    availableVlans.forEach(v => { scopeOptions += `<option value="${v.subnet}">${v.name} (${v.subnet})</option>`; });
    
    let typeOptions = '<option value="custom">Manuel</option>';
    Object.keys(standardFlows).forEach(key => {
        if (key !== 'custom') {
            typeOptions += `<option value="${key}">${key.toUpperCase()}</option>`;
        }
    });

    row.innerHTML = `
        <td><button type="button" class="btn btn-danger btn-sm" onclick="this.closest('tr').remove()">X</button></td>
        <td><select class="form-select form-select-sm mb-1" onchange="selectScope(this, ${idx}, 'source')">${scopeOptions}</select><input type="text" name="flows[${idx}].source_vlan" class="form-control" id="source_vlan_${idx}" readonly placeholder="VLAN auto"></td>
        <td><input type="text" name="flows[${idx}].source_ip" class="form-control" id="source_ip_${idx}" placeholder="IP ou Subnet" list="ciSuggestions" oninput="updateSuggestions(this.value)"></td>
        <td><input type="text" name="flows[${idx}].source_hostname" class="form-control" id="source_hostname_${idx}" placeholder="Hostname" list="ciSuggestions" oninput="updateSuggestions(this.value)"></td>
        <td><select class="form-select form-select-sm mb-1" onchange="selectScope(this, ${idx}, 'target')">${scopeOptions}</select><input type="text" name="flows[${idx}].target_vlan" class="form-control" id="target_vlan_${idx}" readonly placeholder="VLAN auto"></td>
        <td><input type="text" name="flows[${idx}].target_ip" class="form-control" id="target_ip_${idx}" placeholder="IP ou Subnet" list="ciSuggestions" oninput="updateSuggestions(this.value)"></td>
        <td><input type="text" name="flows[${idx}].target_hostname" class="form-control" id="target_hostname_${idx}" placeholder="Hostname" list="ciSuggestions" oninput="updateSuggestions(this.value)"></td>
        <td>
            <select class="form-select" id="flow_type_${idx}" onchange="applyStandardFlow(${idx}, this.value)">
                ${typeOptions}
            </select>
        </td>
        <td>
            <select name="flows[${idx}].protocol" id="protocol_${idx}" class="form-select">
                <option value="TCP">TCP</option>
                <option value="UDP">UDP</option>
                <option value="BOTH">TCP & UDP</option>
                <option value="ICMP">ICMP</option>
            </select>
        </td>
        <td><input type="text" name="flows[${idx}].ports" id="ports_${idx}" class="form-control"></td>
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

function applyStandardFlow(idx, type) {
    const flow = standardFlows[type];
    if (!flow) return;
    document.getElementById(`protocol_${idx}`).value = flow.protocol;
    document.getElementById(`ports_${idx}`).value = flow.ports;
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
            if (ci.vlan_name) v.value = ci.vlan_name;
        }
    } catch (err) { console.error(err); }
}

// Sorting logic
function setupTableSorting() {
    document.querySelectorAll('th.sortable').forEach(th => {
        th.addEventListener('click', () => {
            const table = th.closest('table');
            const tbody = table.querySelector('tbody');
            const rows = Array.from(tbody.querySelectorAll('tr'));
            const columnIdx = Array.from(th.parentNode.children).indexOf(th);
            const isAscending = th.classList.contains('sort-asc');
            
            // Clear other headers
            th.parentNode.querySelectorAll('th').forEach(header => {
                header.classList.remove('sort-asc', 'sort-desc');
            });

            rows.sort((a, b) => {
                let cellA = a.children[columnIdx].textContent.trim();
                let cellB = b.children[columnIdx].textContent.trim();

                // Handle numbers (ports, IDs)
                if (!isNaN(cellA) && !isNaN(cellB) && cellA !== '' && cellB !== '') {
                    return isAscending ? cellB - cellA : cellA - cellB;
                }

                // Handle dates (JJ/MM/AAAA)
                const dateRegex = /^(\d{2})\/(\d{2})\/(\d{4})/;
                if (dateRegex.test(cellA) && dateRegex.test(cellB)) {
                    const dA = new Date(cellA.replace(dateRegex, '$3-$2-$1'));
                    const dB = new Date(cellB.replace(dateRegex, '$3-$2-$1'));
                    return isAscending ? dB - dA : dA - dB;
                }

                // Default string comparison
                return isAscending 
                    ? cellB.localeCompare(cellA, 'fr', {numeric: true}) 
                    : cellA.localeCompare(cellB, 'fr', {numeric: true});
            });

            th.classList.toggle('sort-asc', !isAscending);
            th.classList.toggle('sort-desc', isAscending);
            
            rows.forEach(row => tbody.appendChild(row));
        });
    });
}

// Administration Logic
async function loadStandardFlows() {
    const tbody = document.getElementById('standard-flows-table-body');
    if (!tbody) return;
    tbody.innerHTML = '<tr><td colspan="4" class="text-center">Chargement...</td></tr>';
    try {
        const res = await fetch('/standard-flows');
        if (res.ok) {
            const data = await res.json();
            tbody.innerHTML = '';
            data.forEach(f => {
                const tr = document.createElement('tr');
                tr.innerHTML = `
                    <td>${f.name}</td>
                    <td>${f.protocol}</td>
                    <td>${f.ports}</td>
                    <td>
                        <button class="btn btn-sm btn-warning" onclick='prepareStandardFlowModal(${JSON.stringify(f)})'>Modifier</button>
                        <button class="btn btn-sm btn-danger" onclick="deleteStandardFlow(${f.ID})">Supprimer</button>
                    </td>
                `;
                tbody.appendChild(tr);
            });
        }
    } catch (err) { console.error(err); }
}

function prepareStandardFlowModal(flow) {
    document.getElementById('standardFlowModalLabel').innerText = flow ? 'Modifier le flux' : 'Ajouter un flux';
    document.getElementById('flowId').value = flow ? flow.ID : '';
    document.getElementById('flowName').value = flow ? flow.name : '';
    document.getElementById('flowProtocol').value = flow ? flow.protocol : 'TCP';
    document.getElementById('flowPorts').value = flow ? flow.ports : '';
    const modal = new bootstrap.Modal(document.getElementById('standardFlowModal'));
    modal.show();
}

async function saveStandardFlow() {
    const id = document.getElementById('flowId').value;
    const data = {
        name: document.getElementById('flowName').value,
        protocol: document.getElementById('flowProtocol').value,
        ports: document.getElementById('flowPorts').value
    };
    const method = id ? 'PUT' : 'POST';
    const url = id ? `/standard-flow/${id}` : '/standard-flow';
    const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json', 'X-CSRF-TOKEN': csrfToken },
        body: JSON.stringify(data)
    });
    if (res.ok) {
        bootstrap.Modal.getInstance(document.getElementById('standardFlowModal')).hide();
        loadStandardFlows();
        fetchStandardFlows(); // Update the cache for the request form
    } else alert('Erreur lors de la sauvegarde');
}

async function deleteStandardFlow(id) {
    if (!confirm('Supprimer ce flux standard ?')) return;
    const res = await fetch(`/standard-flow/${id}`, { method: 'DELETE', headers: { 'X-CSRF-TOKEN': csrfToken } });
    if (res.ok) {
        loadStandardFlows();
        fetchStandardFlows();
    }
}

// User / VLAN / CI existing functions...
function prepareVlanModal(vlan) {
    document.getElementById('vlanModalLabel').innerText = vlan ? 'Modifier le VLAN' : 'Ajouter un VLAN';
    document.getElementById('vlanId').value = vlan ? vlan.ID : '';
    document.getElementById('vlanSubnet').value = vlan ? vlan.Subnet : '';
    document.getElementById('vlanName').value = vlan ? vlan.VLAN : '';
    document.getElementById('vlanGateway').value = vlan ? vlan.Gateway : '';
    document.getElementById('vlanDNSServers').value = vlan ? vlan.DNSServers : '';
}

async function saveVlan() {
    const id = document.getElementById('vlanId').value;
    const data = {
        subnet: document.getElementById('vlanSubnet').value,
        vlan: document.getElementById('vlanName').value,
        gateway: document.getElementById('vlanGateway').value,
        dns_servers: document.getElementById('vlanDNSServers').value
    };
    const method = id ? 'PUT' : 'POST';
    const url = id ? `/vlan/${id}` : '/vlan';
    const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json', 'X-CSRF-TOKEN': csrfToken },
        body: JSON.stringify(data)
    });
    if (res.ok) location.reload(); else alert('Erreur lors de la sauvegarde');
}

async function deleteVlan(id) {
    if (!confirm('Supprimer ce VLAN ?')) return;
    const res = await fetch(`/vlan/${id}`, { method: 'DELETE', headers: { 'X-CSRF-TOKEN': csrfToken } });
    if (res.ok) location.reload();
}

async function uploadVlanCSV() {
    const file = document.getElementById('csvFileInput').files[0];
    if (!file) return;
    const formData = new FormData(); formData.append('file', file);
    const res = await fetch('/vlan/import', {method: 'POST', headers: { 'X-CSRF-TOKEN': csrfToken }, body: formData});
    if (res.ok) { const data = await res.json(); alert(data.message); location.reload(); } else alert('Erreur import');
}

function prepareCiModal(ci) {
    document.getElementById('ciModalLabel').innerText = ci ? 'Modifier le CI' : 'Ajouter un CI';
    document.getElementById('ciId').value = ci ? ci.ID : '';
    document.getElementById('ciHostname').value = ci ? ci.Hostname : '';
    document.getElementById('ciIP').value = ci ? ci.IP : '';
    document.getElementById('ciDescription').value = ci ? ci.Description : '';
}

async function saveCi() {
    const id = document.getElementById('ciId').value;
    const data = {
        hostname: document.getElementById('ciHostname').value,
        ip: document.getElementById('ciIP').value,
        description: document.getElementById('ciDescription').value
    };
    const method = id ? 'PUT' : 'POST';
    const url = id ? `/ci/${id}` : '/ci';
    const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json', 'X-CSRF-TOKEN': csrfToken },
        body: JSON.stringify(data)
    });
    if (res.ok) location.reload(); else alert('Erreur lors de la sauvegarde');
}

async function deleteCi(id) {
    if (!confirm('Supprimer ce CI ?')) return;
    const res = await fetch(`/ci/${id}`, { method: 'DELETE', headers: { 'X-CSRF-TOKEN': csrfToken } });
    if (res.ok) location.reload();
}

async function uploadCiCSV() {
    const file = document.getElementById('ciCsvFileInput').files[0];
    if (!file) return;
    const formData = new FormData(); formData.append('file', file);
    const res = await fetch('/ci/import', {method: 'POST', headers: { 'X-CSRF-TOKEN': csrfToken }, body: formData});
    if (res.ok) { const data = await res.json(); alert(data.message); location.reload(); } else alert('Erreur import');
}

function prepareFlowEditModal(id, ruleNumber, status, comment, history) {
    document.getElementById('editFlowId').value = id;
    document.getElementById('editRuleNumber').value = ruleNumber;
    document.getElementById('editStatus').value = status;
    document.getElementById('editComment').value = comment;

    const historyTbody = document.getElementById('flowHistoryTableBody');
    historyTbody.innerHTML = '';
    if (history && history.length > 0) {
        history.forEach(h => {
            const tr = document.createElement('tr');
            // GORM/JSON can use created_at or CreatedAt depending on tags
            const rawDate = h.created_at || h.CreatedAt;
            const dateObj = new Date(rawDate);
            const dateStr = isNaN(dateObj.getTime()) ? 'Date inconnue' : dateObj.toLocaleString('fr-FR');
            
            tr.innerHTML = `
                <td>${dateStr}</td>
                <td><span class="badge ${h.status === 'terminé' ? 'bg-success' : 'bg-secondary'}">${h.status}</span></td>
                <td>${h.actor || 'Système'}</td>
            `;
            historyTbody.appendChild(tr);
        });
    } else {
        historyTbody.innerHTML = '<tr><td colspan="3" class="text-center">Aucun historique</td></tr>';
    }
}

async function saveFlowEdit() {
    const id = document.getElementById('editFlowId').value;
    const data = {
        rule_number: document.getElementById('editRuleNumber').value,
        status: document.getElementById('editStatus').value,
        comment: document.getElementById('editComment').value
    };
    const res = await fetch(`/flow/${id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', 'X-CSRF-TOKEN': csrfToken },
        body: JSON.stringify(data)
    });
    if (res.ok) location.reload(); else alert('Erreur lors de la mise à jour');
}

function prepareUserModal(user) {
    document.getElementById('userModalLabel').innerText = user ? "Modifier l'utilisateur" : "Ajouter un utilisateur";
    document.getElementById('userId').value = user ? user.ID : '';
    document.getElementById('userUsername').value = user ? user.Username : '';
    document.getElementById('userPassword').value = '';
    document.getElementById('userRole').value = user ? user.Role : 'viewer';
}

async function saveUser() {
    const id = document.getElementById('userId').value;
    const data = {
        username: document.getElementById('userUsername').value,
        password: document.getElementById('userPassword').value,
        role: document.getElementById('userRole').value
    };
    const method = id ? 'PUT' : 'POST';
    const url = id ? `/users/${id}` : '/users';
    const res = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json', 'X-CSRF-TOKEN': csrfToken },
        body: JSON.stringify(data)
    });
    if (res.ok) location.reload(); else alert('Erreur lors de la sauvegarde');
}

async function deleteUser(id) {
    if (!confirm('Supprimer cet utilisateur ?')) return;
    const res = await fetch(`/users/${id}`, { method: 'DELETE', headers: { 'X-CSRF-TOKEN': csrfToken } });
    if (res.ok) location.reload();
}

document.addEventListener('DOMContentLoaded', () => {
    fetchStandardFlows();
    setupTableSorting();
    if (document.getElementById('form-body')) addRow();
});
