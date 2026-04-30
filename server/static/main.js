const listEl = document.getElementById('list');
const toastEl = document.getElementById('toast');
let toastTimer;

function toast(msg, timeout) {
    toastEl.innerHTML = msg;
    toastEl.classList.add('show');
    clearTimeout(toastTimer);
    if (!timeout) {
        timeout = 2000;
    }
    if (timeout > 0) {
        toastTimer = setTimeout(() => toastEl.classList.remove('show'), timeout);
    }
}

async function hdlRenameStart(ev) {
    const entryEl = ev.target.closest('.entry');

    const nameEl = entryEl.querySelector('.name');
    nameEl.removeAttribute('readonly');
    nameEl.value = '';
    nameEl.focus({ focusVisible: true });
    nameEl.select();
}

async function hdlRenameDone(ev) {
    const entryEl = ev.target.closest('.entry');
    const id = entryEl.dataset.id;
    const name = entryEl.dataset.name;

    const nameEl = entryEl.querySelector('.name');
    const newName = nameEl.value.trim();

    if (newName == '' || newName == name) {
        nameEl.value = name;
        nameEl.setAttribute('readonly', true);
        return;
    }

    const r = await fetch('/api/accounts/rename', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: id, name: newName }) });
    if (r.ok) {
        toast(`Renamed "${name}" to "${newName}".`);
        reload();
    } else {
        toast('Error: ' + await r.text());
    }
}

async function hdlLoad(ev) {
    const entryEl = ev.target.closest('.entry');
    const id = entryEl.dataset.id;
    const name = entryEl.dataset.name;

    const r = await fetch('/api/accounts/load', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: id }) });
    if (r.ok) {
        toast(`Loaded "${name}".`);
    } else {
        toast('Error: ' + await r.text());
    };
}

async function hdlDelete(ev) {
    const entryEl = ev.target.closest('.entry');
    const id = entryEl.dataset.id;
    const name = entryEl.dataset.name;

    if (!confirm(`Delete "${name}"?`)) return;

    const r = await fetch('/api/accounts/delete', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: id }) });
    if (r.ok) {
        toast(`Deleted "${name}".`);
        reload();
    } else {
        toast('Error: ' + await r.text());
    }
}

async function reload() {
    const resp = await fetch('/api/accounts/list');
    const entries = await resp.json();
    listEl.innerHTML = '';

    for (const e of entries) {
        const id = e.id;
        const name = e.name;

        const entryEl = document.createElement('div');
        entryEl.className = 'entry';
        entryEl.setAttribute('data-id', id);
        entryEl.setAttribute('data-name', name);
        entryEl.innerHTML =
            '<input name="name" class="name" readonly="true" />' +
            '<button class="btn btn-acc load">Load</button>' +
            '<button class="btn btn-acc delete"">Delete</button>';

        const nameEl = entryEl.querySelector('.name');

        nameEl.value = name;
        nameEl.placeholder = name;
        nameEl.addEventListener('dblclick', hdlRenameStart);
        nameEl.addEventListener('focusout', hdlRenameDone);

        const loadEl = entryEl.querySelector('.load');
        loadEl.addEventListener('click', hdlLoad);

        const deleteEl = entryEl.querySelector('.delete');
        deleteEl.addEventListener('click', hdlDelete);

        listEl.appendChild(entryEl);
    }

    const entryEl = document.createElement('div');
    entryEl.className = 'entry';
    entryEl.innerHTML =
        '<input class="name" placeholder="New entry" />' +
        '<button class="btn btn-acc store">Store</button>'

    const nameEl = entryEl.querySelector('.name');

    const storeEl = entryEl.querySelector('.store');
    storeEl.addEventListener('click', async (ev) => {
        console.log('storeEl.click', ev);

        const name = nameEl.value.trim() || 'Unnamed';
        const r = await fetch('/api/accounts/store', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: name }) });
        if (r.ok) {
            toast(`Stored "${name}".`);
            reload();
        } else {
            toast('Error: ' + await r.text());
        }
    });

    listEl.appendChild(entryEl);
}

reload();

async function refreshExpStats() {
    const r = await fetch('/api/exp/stats');
    if (!r.ok) return;
    const data = await r.json();

    const fmtInt = (x) => {
        if (!Number.isInteger(x)) {
            return '-';
        }
        return x.toLocaleString('en-US');
    };

    const fmtDur = (x) => {
        if (!Number.isInteger(x)) {
            return '-';
        }
        const h = Math.floor(x / 3600);
        x -= h * 3600;
        const m = Math.floor(x / 60);
        x -= m * 60;
        const s = x;

        let durStr = '';
        if (h) durStr += `${h}h`;
        if (m) durStr += `${m}m`;
        if (s) durStr += `${s}s`;

        if (!durStr) return '-';
        return durStr;
    };

    document.getElementById('exp-val-level').textContent = fmtInt(data.level);
    document.getElementById('exp-val-total').textContent = fmtInt(data.total_exp);
    document.getElementById('exp-val-remaining').textContent = fmtInt(data.remaining_exp);
    document.getElementById('exp-val-session-delta').textContent = fmtInt(data.session_delta);
    document.getElementById('exp-val-session-duration').textContent = fmtDur(data.session_duration_sec);
    document.getElementById('exp-val-session-rate').textContent = fmtInt(data.session_rate);

    document.querySelectorAll('.exp-value').forEach((x) => {
        if (data.paused) {
            x.classList.add('exp-value-paused');
        } else {
            x.classList.remove('exp-value-paused');
        }
    });

    document.getElementById('exp-btn-run').textContent = data.running ? 'Stop' : 'Start';
    document.getElementById('exp-btn-run').dataset.action = data.running ? '/api/exp/stop' : '/api/exp/start';
    document.getElementById('exp-btn-run').disabled = false;

    document.getElementById('exp-btn-pause').textContent = data.paused ? 'Unpause' : 'Pause';
    document.getElementById('exp-btn-pause').dataset.action = data.paused ? '/api/exp/unpause' : '/api/exp/pause';
    document.getElementById('exp-btn-pause').disabled = false;

    document.getElementById('exp-btn-reset').disabled = !data.running;
    document.getElementById('exp-btn-reset').dataset.action = '/api/exp/reset';

    document.getElementById('exp-container').dataset.autorefresh = Number(data.running && !data.paused);
}

refreshExpStats().then(() => {
    document.querySelectorAll('.btn-exp').forEach((x) => {
        x.addEventListener('click', async () => {
            document.querySelectorAll('.btn-exp').forEach((x) => { x.disabled = true; });
            fetch(x.dataset.action).then(refreshExpStats);
        });
    });
});

setInterval(() => {
    if (Number(document.getElementById('exp-container').dataset.autorefresh)) {
        refreshExpStats();
    }
}, 1000);

try {
    fetch('/api/version').then(async (r) => {
        if (!r.ok) return;
        const json = await r.json()
        const version = json.version;
        document.getElementById('title').innerText = `Tibiantis Assistant ${version}`;
        if (!version.match(/v\d\.\d\.\d/)) return;
        fetch('https://api.github.com/repos/s5i/tassist/releases/latest').then(async (r) => {
            if (!r.ok) return;
            const json = await r.json()
            const latest = json.tag_name;
            if (version == latest) return;
            const url = json.assets[0].browser_download_url;
            toast(`New version available! Download <a href="${url}" class="link">here</a>.`, -1);
        });
    })
} catch { };

// Foreground keepalive.
setInterval(() => { fetch('/api/ping').catch(() => { window.close(); }); }, 1000);

// Background keepalive; a little less intense.
new Worker(window.URL.createObjectURL(new Blob([`setInterval(() => { fetch('${window.location.href}'+'api/ping').catch(); }, 10000);`], { type: "text/javascript" })));
