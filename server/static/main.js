const toast = {
    msg: function (msg, timeout) {
        const toastEl = document.getElementById('toast');
        toastEl.innerHTML = msg;
        toastEl.classList.add('toast-visible');
        clearTimeout(toast.timer);
        if (!timeout) {
            timeout = 2000;
        }
        if (timeout > 0) {
            toast.timer = setTimeout(() => toastEl.classList.remove('toast-visible'), timeout);
        }
    },
    timer: undefined,
};

const ping = {
    run: async function () {
        // Foreground keepalive.
        setInterval(() => { fetch('/api/ping').catch(() => { window.close(); }); }, 1000);

        // Background keepalive; a little less intense.
        new Worker(window.URL.createObjectURL(new Blob([`setInterval(() => { fetch('${window.location.href}'+'api/ping').catch(); }, 10000);`], { type: "text/javascript" })));
    },
};

const version = {
    run: async function () {
        const resp = await fetch('/api/version');
        if (!resp.ok) return;
        const d = await resp.json()
        const version = d.version;

        document.getElementById('title').innerText = `Tibiantis Assistant ${version}`;

        try {
            if (!version.match(/v\d\.\d\.\d/)) return;

            const resp = await fetch('https://api.github.com/repos/s5i/tassist/releases/latest');
            if (!resp.ok) return;
            const d = await resp.json()

            const latest = d.tag_name;
            const url = d.assets[0].browser_download_url;
            if (version != latest) {
                toast.msg(`New version available! Download <a href="${url}" class="link">here</a>.`, -1);
            }
        } catch { }
    },
};

const exp = {
    run: function () {
        document.querySelectorAll('.exp-btn').forEach((x) => {
            x.addEventListener('click', exp.hdlGeneric);
        });

        setInterval(() => {
            if (Number(document.getElementById('exp-container').dataset.autorefresh)) {
                exp.reload();
            }
        }, 1000);

        exp.reload();
    },
    reload: async function () {
        const r = await fetch('/api/exp/stats');
        if (!r.ok) return;
        const d = await r.json();

        document.getElementById('exp-val-level').textContent = exp.fmtInt(d.level);
        document.getElementById('exp-val-total').textContent = exp.fmtInt(d.total_exp);
        document.getElementById('exp-val-remaining').textContent = exp.fmtInt(d.remaining_exp);
        document.getElementById('exp-val-session-delta').textContent = exp.fmtInt(d.session_delta);
        document.getElementById('exp-val-session-duration').textContent = exp.fmtDuration(d.session_duration_sec);
        document.getElementById('exp-val-session-rate').textContent = exp.fmtInt(d.session_rate);

        document.querySelectorAll('.exp-value').forEach((x) => {
            x.classList.remove('exp-value-paused');
            if (d.paused) {
                x.classList.add('exp-value-paused');
            }
        });

        document.getElementById('exp-btn-run').textContent = d.running ? 'Stop' : 'Start';
        document.getElementById('exp-btn-run').dataset.action = d.running ? '/api/exp/stop' : '/api/exp/start';
        document.getElementById('exp-btn-pause').textContent = d.paused ? 'Unpause' : 'Pause';
        document.getElementById('exp-btn-pause').dataset.action = d.paused ? '/api/exp/unpause' : '/api/exp/pause';
        document.getElementById('exp-btn-reset').dataset.action = '/api/exp/reset';
        document.getElementById('exp-container').dataset.autorefresh = Number(d.running && !d.paused);
    },
    fmtInt: function (x) {
        if (!Number.isInteger(x)) {
            return '-';
        }
        return x.toLocaleString('en-US');
    },
    fmtDuration: function (x) {
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
    },
    hdlGeneric: async function (ev) {
        document.querySelectorAll('.exp-btn').forEach((x) => { x.disabled = true; });

        const btnEl = ev.target.closest('.exp-btn');
        fetch(btnEl.dataset.action).then(exp.reload);

        document.querySelectorAll('.exp-btn').forEach((x) => { x.removeAttribute('disabled'); });
    },
};

const acc = {
    run: function () {
        acc.reload();
    },
    reload: async function () {
        const resp = await fetch('/api/accounts/list');
        if (!resp.ok) return;
        const d = await resp.json();

        const accListEl = document.getElementById('acc-list');
        accListEl.innerHTML = '';

        for (const entry of d) {
            const id = entry.id;
            const name = entry.name;

            const entryEl = document.createElement('div');
            accListEl.appendChild(entryEl);
            entryEl.classList.add('acc-entry');
            entryEl.setAttribute('data-id', id);
            entryEl.setAttribute('data-name', name);

            const nameEl = document.createElement('input');
            entryEl.appendChild(nameEl);
            nameEl.classList.add('acc-name');
            nameEl.setAttribute('name', 'name');
            nameEl.setAttribute('readonly', true);
            nameEl.setAttribute('value', name);
            nameEl.setAttribute('placeholder', name);
            nameEl.addEventListener('dblclick', acc.hdlRenameStart);
            nameEl.addEventListener('focusout', acc.hdlRenameDone);

            const loadEl = document.createElement('button');
            entryEl.appendChild(loadEl);
            loadEl.textContent = 'Load';
            loadEl.classList.add('btn', 'acc-btn');
            loadEl.addEventListener('click', acc.hdlLoad);

            const deleteEl = document.createElement('button');
            entryEl.appendChild(deleteEl);
            deleteEl.textContent = 'Delete';
            deleteEl.classList.add('btn', 'acc-btn');
            deleteEl.addEventListener('click', acc.hdlDelete);
        }

        {
            const entryEl = document.createElement('div');
            accListEl.appendChild(entryEl);
            entryEl.classList.add('acc-entry');

            const nameEl = document.createElement('input');
            entryEl.appendChild(nameEl);
            nameEl.classList.add('acc-name');
            nameEl.setAttribute('placeholder', 'New entry');

            const storeEl = document.createElement('button');
            entryEl.appendChild(storeEl);
            storeEl.textContent = 'Store';
            storeEl.classList.add('btn', 'acc-btn');
            storeEl.addEventListener('click', acc.hdlStore);
        }
    },
    hdlRenameStart: async function (ev) {
        const entryEl = ev.target.closest('.acc-entry');
        const nameEl = entryEl.querySelector('.acc-name');

        nameEl.removeAttribute('readonly');
        nameEl.value = '';
        nameEl.focus({ focusVisible: true });
        nameEl.select();
    },
    hdlRenameDone: async function (ev) {
        const entryEl = ev.target.closest('.acc-entry');
        const id = entryEl.dataset.id;
        const name = entryEl.dataset.name;
        const nameEl = entryEl.querySelector('.acc-name');
        const newName = nameEl.value.trim();

        if (newName == '' || newName == name) {
            nameEl.value = name;
            nameEl.setAttribute('readonly', true);
            return;
        }

        const r = await fetch('/api/accounts/rename', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: id, name: newName }) });
        if (r.ok) {
            toast.msg(`Renamed "${name}" to "${newName}".`);
            acc.reload();
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
    hdlLoad: async function (ev) {
        const entryEl = ev.target.closest('.acc-entry');
        const id = entryEl.dataset.id;
        const name = entryEl.dataset.name;

        const r = await fetch('/api/accounts/load', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: id }) });
        if (r.ok) {
            toast.msg(`Loaded "${name}".`);
        } else {
            toast.msg('Error: ' + await r.text());
        };
    },
    hdlDelete: async function (ev) {
        const entryEl = ev.target.closest('.acc-entry');
        const id = entryEl.dataset.id;
        const name = entryEl.dataset.name;

        if (!confirm(`Delete "${name}"?`)) return;

        const r = await fetch('/api/accounts/delete', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: id }) });
        if (r.ok) {
            toast.msg(`Deleted "${name}".`);
            acc.reload();
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
    hdlStore: async function (ev) {
        const entryEl = ev.target.closest('.acc-entry');
        const nameEl = entryEl.querySelector('.acc-name');
        const name = nameEl.value.trim() || 'Unnamed';

        const r = await fetch('/api/accounts/store', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: name }) });
        if (r.ok) {
            toast.msg(`Stored "${name}".`);
            acc.reload();
        } else {
            toast.msg('Error: ' + await r.text());
        }
    },
};

ping.run();
version.run();
exp.run();
acc.run();
