import './style.css';
import './app.css';

import { GetTunnels, AddTunnel, UpdateTunnel, DeleteTunnel, StartTunnel, StopTunnel, GetStatuses, GetSettings, SaveSettings } from '../wailsjs/go/main/App';

// ── State ─────────────────────────────────────────
let tunnels = [];
let statuses = {};
let selectedId = null;
let statusInterval = null;
let currentSettings = {};
let previewTheme = null;

// ── Init ──────────────────────────────────────────
document.querySelector('#app').innerHTML = `
  <div class="header">
    <div class="header-brand">
      <span class="header-icon" style="color:#3fb950;text-shadow:0 0 8px #3fb950;font-family:monospace;font-size:20px;">~</span>
      TunnelDeck
    </div>
    <div class="header-actions">
      <button class="btn btn-secondary btn-sm" id="btn-settings">⚙ Settings</button>
      <button class="btn btn-primary btn-sm" id="btn-add">+ New Tunnel</button>
    </div>
  </div>

  <div class="layout">
    <aside class="sidebar">
      <div class="sidebar-header">Tunnels</div>
      <div class="tunnel-list" id="tunnel-list"></div>
    </aside>
    <main class="main" id="main-panel"></main>
  </div>

  <!-- Add/Edit Modal -->
  <div class="modal-overlay hidden" id="modal-overlay">
    <div class="modal">
      <div class="modal-header">
        <span class="modal-title" id="modal-title">New Tunnel</span>
        <button class="modal-close" id="modal-close">✕</button>
      </div>
      <div class="modal-body">
        <input type="hidden" id="form-id" />
        <div class="form-group">
          <label class="form-label">Name <span class="required">*</span></label>
          <input class="form-input" id="form-name" placeholder="e.g. prod-postgres" />
        </div>
        <div class="form-row">
          <div class="form-group">
            <label class="form-label">SSH Host <span class="required">*</span></label>
            <input class="form-input" id="form-ssh-host" placeholder="192.168.1.1" />
          </div>
          <div class="form-group">
            <label class="form-label">SSH Port</label>
            <input class="form-input" id="form-ssh-port" type="number" value="22" />
          </div>
        </div>
        <div class="form-group">
          <label class="form-label">SSH User <span class="required">*</span></label>
          <input class="form-input" id="form-user" placeholder="ubuntu" />
        </div>

        <div class="form-group">
          <label class="form-label">Auth Type</label>
          <select class="form-select" id="form-auth-type">
            <option value="password">Password</option>
            <option value="key">SSH Key File</option>
          </select>
        </div>
        <div class="auth-section" id="auth-password-section">
          <div class="form-group" style="margin-bottom:0">
            <label class="form-label">Password</label>
            <input class="form-input" id="form-password" type="password" placeholder="••••••••" autocomplete="new-password"/>
          </div>
        </div>
        <div class="auth-section hidden" id="auth-key-section">
          <div class="form-group" style="margin-bottom:0">
            <label class="form-label">Private Key Path</label>
            <input class="form-input" id="form-key-path" placeholder="C:\\Users\\you\\.ssh\\id_rsa" />
            <div class="form-hint">Absolute path to your .pem or id_rsa file</div>
          </div>
        </div>

        <div class="divider"></div>
        <label class="toggle-switch" style="margin-bottom:0">
          <input type="checkbox" id="form-use-bastion" />
          <span class="toggle-track"></span>
          Jump Host / Bastion
        </label>
        <div id="bastion-section" class="hidden" style="margin-top:12px">
          <div class="auth-section" style="display:flex;flex-direction:column;gap:12px">
            <div class="form-row" style="margin-bottom:0">
              <div class="form-group" style="margin-bottom:0">
                <label class="form-label">Bastion Host <span class="required">*</span></label>
                <input class="form-input" id="form-bastion-host" placeholder="bastion.example.com" />
              </div>
              <div class="form-group" style="margin-bottom:0">
                <label class="form-label">Bastion Port</label>
                <input class="form-input" id="form-bastion-port" type="number" value="22" />
              </div>
            </div>
            <div class="form-group" style="margin-bottom:0">
              <label class="form-label">Bastion User <span class="required">*</span></label>
              <input class="form-input" id="form-bastion-user" placeholder="ec2-user" />
            </div>
            <div class="form-group" style="margin-bottom:0">
              <label class="form-label">Bastion Auth Type</label>
              <select class="form-select" id="form-bastion-auth-type">
                <option value="password">Password</option>
                <option value="key">SSH Key File</option>
              </select>
            </div>
            <div id="bastion-auth-password-section">
              <div class="form-group" style="margin-bottom:0">
                <label class="form-label">Bastion Password</label>
                <input class="form-input" id="form-bastion-password" type="password" placeholder="••••••••" autocomplete="new-password"/>
              </div>
            </div>
            <div id="bastion-auth-key-section" class="hidden">
              <div class="form-group" style="margin-bottom:0">
                <label class="form-label">Bastion Key Path</label>
                <input class="form-input" id="form-bastion-key-path" placeholder="C:\\Users\\you\\.ssh\\id_rsa" />
              </div>
            </div>
          </div>
        </div>

        <div class="divider"></div>
        <div class="section-title">Port Forwarding</div>
        <div class="form-row-3">
          <div class="form-group">
            <label class="form-label">Remote Host <span class="required">*</span></label>
            <input class="form-input" id="form-remote-host" placeholder="localhost" />
          </div>
          <div class="form-group">
            <label class="form-label">Remote Port <span class="required">*</span></label>
            <input class="form-input" id="form-remote-port" type="number" placeholder="5432" />
          </div>
          <div class="form-group">
            <label class="form-label">Local Port <span class="required">*</span></label>
            <input class="form-input" id="form-local-port" type="number" placeholder="5432" />
          </div>
        </div>
        <div class="form-hint">Traffic to <code style="color:var(--blue)">127.0.0.1:[local]</code> will forward to <code style="color:var(--blue)">[remote host]:[remote port]</code> via SSH</div>

        <div class="divider"></div>
        <label class="toggle-switch" style="margin-bottom:0">
          <input type="checkbox" id="form-auto-start" />
          <span class="toggle-track"></span>
          Auto-start on launch
        </label>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" id="btn-cancel">Cancel</button>
        <button class="btn btn-primary" id="btn-save">Save Tunnel</button>
      </div>
    </div>
  </div>

  <!-- Settings Modal -->
  <div class="modal-overlay hidden" id="settings-overlay">
    <div class="modal settings-modal">
      <div class="modal-header">
        <span class="modal-title">Settings</span>
        <button class="modal-close" id="settings-close">✕</button>
      </div>
      <div class="modal-body">

        <div class="settings-section-title">Behaviour</div>
        <div class="settings-row">
          <div class="settings-row-info">
            <div class="settings-row-label">Auto-reconnect</div>
            <div class="settings-row-desc">Automatically reconnect tunnels when the connection drops</div>
          </div>
          <label class="toggle-switch">
            <input type="checkbox" id="s-auto-reconnect" />
            <span class="toggle-track"></span>
          </label>
        </div>
        <div class="settings-row" id="s-keepalive-row">
          <div class="settings-row-info">
            <div class="settings-row-label">Keepalive interval</div>
            <div class="settings-row-desc">How often to ping the connection — takes effect on next tunnel start</div>
          </div>
          <div class="settings-row-control">
            <input class="form-input" id="s-keepalive" type="number" min="5" max="300" style="width:64px;text-align:center" />
            <span class="settings-unit">sec</span>
          </div>
        </div>
        <div class="settings-row">
          <div class="settings-row-info">
            <div class="settings-row-label">Launch at login</div>
            <div class="settings-row-desc">Start TunnelDeck automatically when you log in</div>
          </div>
          <label class="toggle-switch">
            <input type="checkbox" id="s-start-on-boot" />
            <span class="toggle-track"></span>
          </label>
        </div>

        <div class="divider"></div>
        <div class="settings-section-title">Appearance</div>
        <div class="settings-row">
          <div class="settings-row-info">
            <div class="settings-row-label">Colour scheme</div>
            <div class="settings-row-desc">Choose your preferred theme</div>
          </div>
          <div class="theme-picker" id="theme-picker">
            <button class="theme-opt" data-theme="dark">Dark</button>
            <button class="theme-opt" data-theme="system">System</button>
            <button class="theme-opt" data-theme="light">Light</button>
          </div>
        </div>

        <div class="divider"></div>
        <div class="settings-section-title">Connection Defaults</div>
        <div class="settings-row">
          <div class="settings-row-info">
            <div class="settings-row-label">SSH port</div>
          </div>
          <div class="settings-row-control">
            <input class="form-input" id="s-default-port" type="number" style="width:72px;text-align:center" />
          </div>
        </div>
        <div class="settings-row">
          <div class="settings-row-info">
            <div class="settings-row-label">SSH user</div>
          </div>
          <div class="settings-row-control">
            <input class="form-input" id="s-default-user" placeholder="ubuntu" style="width:160px" />
          </div>
        </div>
        <div class="settings-row">
          <div class="settings-row-info">
            <div class="settings-row-label">Default key path</div>
            <div class="settings-row-desc">Pre-fills the key path when creating a new tunnel</div>
          </div>
          <div class="settings-row-control">
            <input class="form-input" id="s-default-key" placeholder="~/.ssh/id_rsa" style="width:200px" />
          </div>
        </div>

      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" id="s-cancel">Cancel</button>
        <button class="btn btn-primary" id="s-save">Save</button>
      </div>
    </div>
  </div>

  <!-- Toast container -->
  <div class="toast-container" id="toast-container"></div>
`;

// ── Event Wiring ──────────────────────────────────
document.getElementById('btn-add').addEventListener('click', () => openModal(null));
document.getElementById('btn-settings').addEventListener('click', openSettings);
document.getElementById('modal-close').addEventListener('click', closeModal);
document.getElementById('btn-cancel').addEventListener('click', closeModal);
document.getElementById('btn-save').addEventListener('click', saveTunnel);
document.getElementById('modal-overlay').addEventListener('click', (e) => {
  if (e.target === document.getElementById('modal-overlay')) closeModal();
});

document.getElementById('settings-close').addEventListener('click', closeSettings);
document.getElementById('s-cancel').addEventListener('click', closeSettings);
document.getElementById('s-save').addEventListener('click', saveSettings);
document.getElementById('settings-overlay').addEventListener('click', (e) => {
  if (e.target === document.getElementById('settings-overlay')) closeSettings();
});

document.getElementById('form-auth-type').addEventListener('change', (e) => {
  document.getElementById('auth-password-section').classList.toggle('hidden', e.target.value === 'key');
  document.getElementById('auth-key-section').classList.toggle('hidden', e.target.value === 'password');
});

document.getElementById('form-use-bastion').addEventListener('change', (e) => {
  document.getElementById('bastion-section').classList.toggle('hidden', !e.target.checked);
});

document.getElementById('form-bastion-auth-type').addEventListener('change', (e) => {
  document.getElementById('bastion-auth-password-section').classList.toggle('hidden', e.target.value === 'key');
  document.getElementById('bastion-auth-key-section').classList.toggle('hidden', e.target.value === 'password');
});

document.getElementById('s-auto-reconnect').addEventListener('change', (e) => {
  document.getElementById('s-keepalive-row').classList.toggle('dimmed', !e.target.checked);
});

document.querySelectorAll('.theme-opt').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.theme-opt').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    previewTheme = btn.dataset.theme;
    applyTheme(previewTheme);
  });
});

// ── Theme ──────────────────────────────────────────
function applyTheme(theme) {
  document.documentElement.setAttribute('data-theme', theme || 'dark');
}

// ── Settings ──────────────────────────────────────
function openSettings() {
  const s = currentSettings;
  document.getElementById('s-auto-reconnect').checked = s.autoReconnect !== false;
  document.getElementById('s-keepalive').value = s.keepaliveSeconds || 15;
  document.getElementById('s-start-on-boot').checked = !!s.startOnBoot;
  document.getElementById('s-default-port').value = s.defaultSshPort || 22;
  document.getElementById('s-default-user').value = s.defaultSshUser || '';
  document.getElementById('s-default-key').value = s.defaultKeyPath || '';

  const theme = s.theme || 'dark';
  document.querySelectorAll('.theme-opt').forEach(btn => {
    btn.classList.toggle('active', btn.dataset.theme === theme);
  });

  document.getElementById('s-keepalive-row').classList.toggle('dimmed', s.autoReconnect === false);

  previewTheme = null;
  document.getElementById('settings-overlay').classList.remove('hidden');
}

function closeSettings() {
  if (previewTheme !== null) {
    applyTheme(currentSettings.theme || 'dark');
    previewTheme = null;
  }
  document.getElementById('settings-overlay').classList.add('hidden');
}

async function saveSettings() {
  const theme = document.querySelector('.theme-opt.active')?.dataset.theme || 'dark';
  const s = {
    autoReconnect: document.getElementById('s-auto-reconnect').checked,
    keepaliveSeconds: parseInt(document.getElementById('s-keepalive').value) || 15,
    startOnBoot: document.getElementById('s-start-on-boot').checked,
    theme,
    defaultSshPort: parseInt(document.getElementById('s-default-port').value) || 22,
    defaultSshUser: document.getElementById('s-default-user').value.trim(),
    defaultKeyPath: document.getElementById('s-default-key').value.trim(),
  };
  try {
    await SaveSettings(s);
    currentSettings = s;
    applyTheme(s.theme);
    previewTheme = null;
    document.getElementById('settings-overlay').classList.add('hidden');
    toast('Settings saved', 'success');
  } catch (e) {
    toast('Failed to save settings: ' + e, 'error');
  }
}

// ── Load & Render ─────────────────────────────────
async function loadTunnels() {
  try {
    tunnels = await GetTunnels() || [];
    await refreshStatuses();
    renderSidebar();
    renderMain();
  } catch (e) {
    console.error('Failed to load tunnels:', e);
  }
}

async function refreshStatuses() {
  if (tunnels.length === 0) return;
  try {
    const list = await GetStatuses();
    statuses = {};
    for (const s of list) statuses[s.id] = s;
  } catch (e) {}
}

function renderSidebar() {
  const el = document.getElementById('tunnel-list');
  if (tunnels.length === 0) {
    el.innerHTML = `<div class="empty-sidebar">No tunnels yet.<br/>Click <strong>+ New Tunnel</strong><br/>to get started.</div>`;
    return;
  }
  el.innerHTML = tunnels.map(t => {
    const s = statuses[t.id] || {};
    const dotClass = s.active ? (s.reconnecting ? 'reconnecting' : 'on') : 'off';
    const sel = t.id === selectedId ? 'selected' : '';
    return `
      <div class="tunnel-item ${sel}" data-id="${t.id}" onclick="selectTunnel('${t.id}')">
        <div class="status-dot ${dotClass}"></div>
        <div class="tunnel-item-info">
          <div class="tunnel-item-name">${esc(t.name)}</div>
          <div class="tunnel-item-sub">127.0.0.1:${t.localPort} → ${esc(t.remoteHost)}:${t.remotePort}</div>
        </div>
      </div>`;
  }).join('');
}

function renderMain() {
  const panel = document.getElementById('main-panel');
  if (!selectedId) {
    panel.innerHTML = `
      <div class="welcome">
        <div class="welcome-icon" style="color:#3fb950;text-shadow:0 0 16px #3fb950;font-family:monospace;font-size:64px;opacity:1;">~</div>
        <h2>TunnelDeck</h2>
        <p>Select a tunnel from the sidebar or create a new one.</p>
      </div>`;
    return;
  }
  const t = tunnels.find(x => x.id === selectedId);
  if (!t) { selectedId = null; renderMain(); return; }
  const s = statuses[t.id] || {};

  const dotClass = s.active ? (s.reconnecting ? 'reconnecting' : 'on') : 'off';

  const bastionHop = t.bastionHost ? `
        <div class="forward-box">
          <div class="forward-box-label">Bastion</div>
          <div class="forward-box-value">${esc(t.bastionHost)}:${t.bastionPort || 22}</div>
        </div>
        <div class="forward-arrow">→</div>` : '';

  panel.innerHTML = `
    <div class="detail">
      <div class="detail-header">
        <div class="detail-title-group">
          <div class="detail-status-dot ${dotClass}"></div>
          <div>
            <div class="detail-name">${esc(t.name)}</div>
            ${s.active ? `<div class="detail-uptime">↑ ${s.uptime || '0s'}</div>` : ''}
          </div>
        </div>
        <div class="detail-actions">
          ${s.active
            ? `<button class="btn btn-stop" onclick="stopTunnel('${t.id}')">⏹ Stop</button>`
            : `<button class="btn btn-primary" onclick="startTunnel('${t.id}')">▶ Start</button>`
          }
          <button class="btn btn-secondary" onclick="editTunnel('${t.id}')">✎ Edit</button>
          <button class="btn btn-danger" onclick="deleteTunnel('${t.id}')">✕</button>
        </div>
      </div>

      <div class="forward-vis">
        <div class="forward-box">
          <div class="forward-box-label">Local</div>
          <div class="forward-box-value">127.0.0.1:${t.localPort}</div>
        </div>
        <div class="forward-arrow">→</div>
        ${bastionHop}
        <div class="forward-box">
          <div class="forward-box-label">SSH Server</div>
          <div class="forward-box-value">${esc(t.sshHost)}:${t.sshPort}</div>
        </div>
        <div class="forward-arrow">→</div>
        <div class="forward-box">
          <div class="forward-box-label">Destination</div>
          <div class="forward-box-value">${esc(t.remoteHost)}:${t.remotePort}</div>
        </div>
      </div>

      <div class="cards">
        <div class="card">
          <div class="card-title">SSH Connection</div>
          <div class="card-row">
            <span class="card-label">Host</span>
            <span class="card-value">${esc(t.sshHost)}</span>
          </div>
          <div class="card-row">
            <span class="card-label">Port</span>
            <span class="card-value">${t.sshPort}</span>
          </div>
          <div class="card-row">
            <span class="card-label">User</span>
            <span class="card-value">${esc(t.user)}</span>
          </div>
          <div class="card-row">
            <span class="card-label">Auth</span>
            <span class="card-value"><span class="badge badge-blue">${t.authType}</span></span>
          </div>
          ${t.bastionHost ? `
          <div class="card-row">
            <span class="card-label">Bastion</span>
            <span class="card-value">${esc(t.bastionUser)}@${esc(t.bastionHost)}:${t.bastionPort || 22}</span>
          </div>` : ''}
        </div>
        <div class="card">
          <div class="card-title">Status</div>
          <div class="card-row">
            <span class="card-label">State</span>
            <span class="card-value">
              ${s.active
                ? (s.reconnecting
                    ? '<span class="badge badge-yellow">↻ Reconnecting</span>'
                    : '<span class="badge badge-green">● Connected</span>')
                : '<span class="badge badge-gray">○ Stopped</span>'
              }
            </span>
          </div>
          ${s.active ? `
          <div class="card-row">
            <span class="card-label">Uptime</span>
            <span class="card-value">${s.uptime}</span>
          </div>` : ''}
          <div class="card-row">
            <span class="card-label">Local Bind</span>
            <span class="card-value">127.0.0.1:${t.localPort}</span>
          </div>
          <div class="card-row">
            <span class="card-label">Remote Target</span>
            <span class="card-value">${esc(t.remoteHost)}:${t.remotePort}</span>
          </div>
        </div>
      </div>
    </div>`;
}

// ── Actions ───────────────────────────────────────
window.selectTunnel = function(id) {
  selectedId = id;
  renderSidebar();
  renderMain();
};

window.startTunnel = async function(id) {
  try {
    await StartTunnel(id);
    await refreshStatuses();
    renderSidebar();
    renderMain();
    toast('Tunnel started', 'success');
  } catch (e) {
    toast('Failed to start: ' + e, 'error');
  }
};

window.stopTunnel = async function(id) {
  try {
    await StopTunnel(id);
    await refreshStatuses();
    renderSidebar();
    renderMain();
    toast('Tunnel stopped', 'info');
  } catch (e) {
    toast('Failed to stop: ' + e, 'error');
  }
};

window.editTunnel = function(id) {
  const t = tunnels.find(x => x.id === id);
  if (t) openModal(t);
};

window.deleteTunnel = async function(id) {
  const t = tunnels.find(x => x.id === id);
  if (!t) return;
  if (!confirm(`Delete tunnel "${t.name}"?`)) return;
  try {
    await DeleteTunnel(id);
    if (selectedId === id) selectedId = null;
    await loadTunnels();
    toast('Tunnel deleted', 'info');
  } catch (e) {
    toast('Failed to delete: ' + e, 'error');
  }
};

// ── Modal ─────────────────────────────────────────
function openModal(tunnel) {
  document.getElementById('modal-title').textContent = tunnel ? 'Edit Tunnel' : 'New Tunnel';
  document.getElementById('form-id').value = tunnel?.id || '';
  document.getElementById('form-name').value = tunnel?.name || '';
  document.getElementById('form-ssh-host').value = tunnel?.sshHost || '';

  // For new tunnels, pre-fill connection defaults from settings
  const isNew = !tunnel;
  document.getElementById('form-ssh-port').value = tunnel?.sshPort ?? (currentSettings.defaultSshPort || 22);
  document.getElementById('form-user').value = tunnel?.user ?? (isNew ? (currentSettings.defaultSshUser || '') : '');

  const hasDefaultKey = isNew && currentSettings.defaultKeyPath;
  const authType = tunnel?.authType || (hasDefaultKey ? 'key' : 'password');
  document.getElementById('form-auth-type').value = authType;
  document.getElementById('form-password').value = tunnel?.password || '';
  document.getElementById('form-key-path').value = tunnel?.keyPath ?? (isNew ? (currentSettings.defaultKeyPath || '') : '');

  document.getElementById('auth-password-section').classList.toggle('hidden', authType === 'key');
  document.getElementById('auth-key-section').classList.toggle('hidden', authType === 'password');

  document.getElementById('form-remote-host').value = tunnel?.remoteHost || 'localhost';
  document.getElementById('form-remote-port').value = tunnel?.remotePort || '';
  document.getElementById('form-local-port').value = tunnel?.localPort || '';

  // Bastion fields
  const useBastion = !!tunnel?.bastionHost;
  document.getElementById('form-use-bastion').checked = useBastion;
  document.getElementById('bastion-section').classList.toggle('hidden', !useBastion);
  document.getElementById('form-bastion-host').value = tunnel?.bastionHost || '';
  document.getElementById('form-bastion-port').value = tunnel?.bastionPort || 22;
  document.getElementById('form-bastion-user').value = tunnel?.bastionUser || '';
  document.getElementById('form-bastion-auth-type').value = tunnel?.bastionAuthType || 'password';
  document.getElementById('form-bastion-password').value = tunnel?.bastionPassword || '';
  document.getElementById('form-bastion-key-path').value = tunnel?.bastionKeyPath || '';

  const bastionAuthType = document.getElementById('form-bastion-auth-type').value;
  document.getElementById('bastion-auth-password-section').classList.toggle('hidden', bastionAuthType === 'key');
  document.getElementById('bastion-auth-key-section').classList.toggle('hidden', bastionAuthType === 'password');

  document.getElementById('form-auto-start').checked = !!tunnel?.autoStart;

  document.getElementById('modal-overlay').classList.remove('hidden');
  document.getElementById('form-name').focus();
}

function closeModal() {
  document.getElementById('modal-overlay').classList.add('hidden');
}

async function saveTunnel() {
  const id = document.getElementById('form-id').value;
  const authType = document.getElementById('form-auth-type').value;
  const useBastion = document.getElementById('form-use-bastion').checked;
  const bastionAuthType = document.getElementById('form-bastion-auth-type').value;

  const cfg = {
    id,
    name: document.getElementById('form-name').value.trim(),
    sshHost: document.getElementById('form-ssh-host').value.trim(),
    sshPort: parseInt(document.getElementById('form-ssh-port').value) || 22,
    user: document.getElementById('form-user').value.trim(),
    authType,
    password: authType === 'password' ? document.getElementById('form-password').value : '',
    keyPath: authType === 'key' ? document.getElementById('form-key-path').value.trim() : '',
    remoteHost: document.getElementById('form-remote-host').value.trim() || 'localhost',
    remotePort: parseInt(document.getElementById('form-remote-port').value),
    localPort: parseInt(document.getElementById('form-local-port').value),
    autoStart: document.getElementById('form-auto-start').checked,
    bastionHost: useBastion ? document.getElementById('form-bastion-host').value.trim() : '',
    bastionPort: useBastion ? parseInt(document.getElementById('form-bastion-port').value) || 22 : 0,
    bastionUser: useBastion ? document.getElementById('form-bastion-user').value.trim() : '',
    bastionAuthType: useBastion ? bastionAuthType : '',
    bastionPassword: useBastion && bastionAuthType === 'password' ? document.getElementById('form-bastion-password').value : '',
    bastionKeyPath: useBastion && bastionAuthType === 'key' ? document.getElementById('form-bastion-key-path').value.trim() : '',
  };

  if (!cfg.name || !cfg.sshHost || !cfg.user || !cfg.remotePort || !cfg.localPort) {
    toast('Please fill in all required fields', 'error');
    return;
  }
  if (useBastion && (!cfg.bastionHost || !cfg.bastionUser)) {
    toast('Please fill in bastion host and user', 'error');
    return;
  }

  try {
    if (id) {
      await UpdateTunnel(cfg);
      toast('Tunnel updated', 'success');
    } else {
      const created = await AddTunnel(cfg);
      selectedId = created.id;
      toast('Tunnel created', 'success');
    }
    closeModal();
    await loadTunnels();
  } catch (e) {
    toast('Save failed: ' + e, 'error');
  }
}

// ── Toast ─────────────────────────────────────────
function toast(msg, type = 'info') {
  const icons = { success: '✓', error: '✕', info: 'ℹ' };
  const el = document.createElement('div');
  el.className = `toast toast-${type}`;
  el.innerHTML = `<span>${icons[type]}</span><span>${esc(msg)}</span>`;
  document.getElementById('toast-container').appendChild(el);
  setTimeout(() => el.remove(), 3500);
}

// ── Helpers ───────────────────────────────────────
function esc(str) {
  return String(str || '').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

// ── Boot ──────────────────────────────────────────
async function init() {
  try {
    currentSettings = await GetSettings();
  } catch (e) {
    currentSettings = { theme: 'dark', autoReconnect: true, keepaliveSeconds: 15, defaultSshPort: 22 };
  }
  applyTheme(currentSettings.theme);
  await loadTunnels();

  statusInterval = setInterval(async () => {
    await refreshStatuses();
    renderSidebar();
    const panel = document.getElementById('main-panel');
    if (selectedId && panel.querySelector('.detail')) {
      renderMain();
    }
  }, 3000);
}

init();
