// ============================================================
// PayRoute Dashboard — app.js
// ============================================================

const state = {
    transactions: [],
    currentOrderId: null,
    selectedMethod: 'CARD',
};

// ---- TABS ----
document.querySelectorAll('.nav-item').forEach(btn => {
    btn.addEventListener('click', () => {
        const tab = btn.dataset.tab;
        document.querySelectorAll('.nav-item').forEach(b => b.classList.remove('active'));
        document.querySelectorAll('.tab-panel').forEach(p => p.classList.remove('active'));
        btn.classList.add('active');
        document.getElementById('tab-' + tab).classList.add('active');
        if (tab === 'history') renderTxTable();
        if (tab === 'health') checkHealth();
        if (tab === 'webhook') updateWebhookPreview();
    });
});

// ---- IDEMPOTENCY KEY ----
function newKey() { return crypto.randomUUID(); }
const keyInput = document.getElementById('idempotency-key');
keyInput.value = newKey();
document.getElementById('regen-key').addEventListener('click', () => { keyInput.value = newKey(); });

// ---- CURRENCY PREFIX ----
const prefixMap = { INR: '₹', USD: '$', EUR: '€', GBP: '£' };
document.getElementById('currency').addEventListener('change', e => {
    document.getElementById('currency-prefix').textContent = prefixMap[e.target.value] || '';
});

// ---- METHOD PICKER ----
document.querySelectorAll('.method-btn').forEach(btn => {
    btn.addEventListener('click', () => {
        document.querySelectorAll('.method-btn').forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        state.selectedMethod = btn.dataset.method;
    });
});

// ---- JSON SYNTAX HIGHLIGHT ----
function highlight(json) {
    if (typeof json !== 'string') json = JSON.stringify(json, null, 2);
    return json
        .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
        .replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+\.?\d*(?:[eE][+\-]?\d+)?)/g, m => {
            if (/^"/.test(m)) {
                if (/:$/.test(m)) return `<span class="json-key">${m}</span>`;
                return `<span class="json-str">${m}</span>`;
            }
            if (/true|false/.test(m)) return `<span class="json-bool">${m}</span>`;
            if (/null/.test(m)) return `<span class="json-null">${m}</span>`;
            return `<span class="json-num">${m}</span>`;
        });
}

function setResponse(el, statusEl, data, ok) {
    el.innerHTML = highlight(typeof data === 'string' ? data : JSON.stringify(data, null, 2));
    if (statusEl) {
        statusEl.textContent = ok ? '200 OK' : 'Error';
        statusEl.className = 'response-status ' + (ok ? 'ok' : 'err');
    }
}

function showBanner(text, type) {
    const b = document.getElementById('result-banner');
    b.textContent = text;
    b.className = 'result-banner ' + type;
    b.classList.remove('hidden');
    setTimeout(() => b.classList.add('hidden'), 6000);
}

// ---- STATUS CHIP ----
function chipHtml(status) {
    const cls = status?.toLowerCase() || 'pending';
    return `<span class="status-chip ${cls}">${status || '—'}</span>`;
}

// ---- CREATE PAYMENT ----
document.getElementById('payment-form').addEventListener('submit', async e => {
    e.preventDefault();

    const btn = document.getElementById('submit-btn');
    const btnText = btn.querySelector('.btn-text');
    const loader = btn.querySelector('.btn-loader');
    btn.disabled = true;
    btnText.classList.add('hidden');
    loader.classList.remove('hidden');

    const amount = parseInt(document.getElementById('amount').value, 10);
    const currency = document.getElementById('currency').value;
    const email = document.getElementById('email').value;
    const idempotencyKey = keyInput.value;

    try {
        const resp = await fetch('/api/payments', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Idempotency-Key': idempotencyKey,
            },
            body: JSON.stringify({
                amount,
                currency,
                paymentMethod: state.selectedMethod,
                customerEmail: email,
            }),
        });

        const data = await resp.json();
        const statusEl = document.getElementById('res-status');
        setResponse(document.getElementById('response-json'), statusEl, data, resp.ok);

        if (resp.ok && data.success) {
            const tx = data.data.transaction;
            state.currentOrderId = tx.orderId;
            state.transactions.unshift(tx);

            showBanner(`✓ Payment Intent created: ${data.data.hyperswitchPaymentId}`, 'success');

            // Show reconcile panel
            const panel = document.getElementById('reconcile-section');
            panel.classList.remove('hidden');
            document.getElementById('reconcile-order-id').textContent = tx.orderId;
            document.getElementById('reconcile-status-chip').outerHTML =
                `<span class="status-chip ${tx.status}" id="reconcile-status-chip">${tx.status}</span>`;

            // New idempotency key for next request
            keyInput.value = newKey();
        } else {
            showBanner(`Error: ${data.error || 'Failed to create payment'}`, 'error');
        }
    } catch (err) {
        showBanner(`Network Error: ${err.message}`, 'error');
        setResponse(document.getElementById('response-json'), document.getElementById('res-status'), { error: err.message }, false);
    } finally {
        btn.disabled = false;
        btnText.classList.remove('hidden');
        loader.classList.add('hidden');
    }
});

// ---- RECONCILE ----
document.getElementById('reconcile-btn').addEventListener('click', async () => {
    if (!state.currentOrderId) return;
    const btn = document.getElementById('reconcile-btn');
    btn.disabled = true;
    btn.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="13" height="13" style="animation:spin 0.7s linear infinite"><path d="M23 4v6h-6M1 20v-6h6"/><path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/></svg> Checking…`;

    try {
        const resp = await fetch(`/api/payments/${state.currentOrderId}/reconcile`);
        const data = await resp.json();

        if (resp.ok && data.success) {
            const chipEl = document.getElementById('reconcile-status-chip');
            chipEl.className = `status-chip ${data.currentStatus}`;
            chipEl.textContent = data.currentStatus;
            showBanner(`✓ ${data.message}`, 'success');

            // Update in local tx list
            const tx = state.transactions.find(t => t.orderId === state.currentOrderId);
            if (tx) tx.status = data.currentStatus;
        } else {
            showBanner(`Reconciliation Error: ${data.error}`, 'error');
        }
    } catch (err) {
        showBanner(`Network Error: ${err.message}`, 'error');
    } finally {
        btn.disabled = false;
        btn.innerHTML = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M23 4v6h-6M1 20v-6h6"/><path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/></svg> Reconcile`;
    }
});

// ---- TRANSACTIONS TABLE ----
function renderTxTable(filter = '') {
    const tbody = document.getElementById('tx-tbody');
    const txs = state.transactions.filter(t =>
        !filter ||
        t.orderId?.toLowerCase().includes(filter) ||
        t.status?.toLowerCase().includes(filter) ||
        t.hyperswitchPaymentId?.toLowerCase().includes(filter)
    );

    if (txs.length === 0) {
        tbody.innerHTML = `<tr class="tx-empty-row"><td colspan="7">${filter ? 'No matching transactions.' : 'No transactions yet.'}</td></tr>`;
        return;
    }

    tbody.innerHTML = txs.map(tx => `
        <tr>
            <td><span class="tx-order-id">${tx.orderId || '—'}</span></td>
            <td><span class="tx-amount">${tx.currency || ''} ${(tx.amount / 100).toFixed(2)}</span></td>
            <td>${tx.paymentMethod || '—'}</td>
            <td>${chipHtml(tx.status)}</td>
            <td><span class="tx-hs-id" title="${tx.hyperswitchPaymentId || ''}">${tx.hyperswitchPaymentId || '—'}</span></td>
            <td style="font-family:var(--font-mono);font-size:11px;color:var(--text-muted)">${tx.createdAt ? new Date(tx.createdAt).toLocaleTimeString() : '—'}</td>
            <td>
                <button class="btn-reconcile" onclick="reconcileRow('${tx.orderId}', this)">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M23 4v6h-6M1 20v-6h6"/><path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/></svg>
                    Reconcile
                </button>
            </td>
        </tr>
    `).join('');
}

async function reconcileRow(orderId, btn) {
    btn.disabled = true;
    try {
        const resp = await fetch(`/api/payments/${orderId}/reconcile`);
        const data = await resp.json();
        if (resp.ok && data.success) {
            const tx = state.transactions.find(t => t.orderId === orderId);
            if (tx) tx.status = data.currentStatus;
            renderTxTable(document.getElementById('tx-search').value.trim().toLowerCase());
        }
    } catch (err) { console.error(err); }
    finally { btn.disabled = false; }
}
window.reconcileRow = reconcileRow;

document.getElementById('tx-search').addEventListener('input', e => {
    renderTxTable(e.target.value.trim().toLowerCase());
});

document.getElementById('tx-clear').addEventListener('click', () => {
    if (confirm('Clear all session transactions?')) {
        state.transactions = [];
        renderTxTable();
    }
});

// ---- WEBHOOK SIMULATOR ----
function buildWebhookPayload() {
    const eventType = document.getElementById('wh-event-type').value;
    const paymentId = document.getElementById('wh-payment-id').value || 'pay_demo_' + Date.now();
    const statusMap = {
        'payment_intent.succeeded': 'succeeded',
        'payment_intent.payment_failed': 'failed',
        'payment_intent.processing': 'processing',
        'refund.succeeded': 'succeeded',
    };
    return {
        type: eventType,
        data: {
            object: {
                payment_id: paymentId,
                status: statusMap[eventType] || 'unknown',
                error_message: eventType.includes('failed') ? 'Insufficient funds' : null,
            },
        },
    };
}

function updateWebhookPreview() {
    const payload = buildWebhookPayload();
    document.getElementById('wh-preview').innerHTML = highlight(JSON.stringify(payload, null, 2));
}

document.getElementById('wh-event-type').addEventListener('change', updateWebhookPreview);
document.getElementById('wh-payment-id').addEventListener('input', updateWebhookPreview);

document.getElementById('wh-send-btn').addEventListener('click', async () => {
    const btn = document.getElementById('wh-send-btn');
    btn.disabled = true;
    const payload = buildWebhookPayload();
    const resEl = document.getElementById('wh-response');
    const statusEl = document.getElementById('wh-res-status');

    try {
        const resp = await fetch('/api/webhooks', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
        });
        const text = await resp.text();
        let data;
        try { data = JSON.parse(text); } catch { data = text; }
        setResponse(resEl, statusEl, data, resp.ok);
    } catch (err) {
        setResponse(resEl, statusEl, { error: err.message }, false);
    } finally {
        btn.disabled = false;
    }
});

// ---- HEALTH CHECK ----
async function checkHealth() {
    const setDot = (id, ok) => {
        const d = document.getElementById('hd-' + id);
        if (d) { d.className = 'health-dot ' + (ok ? 'ok' : 'err'); }
    };
    const setSt = (id, txt) => {
        const el = document.getElementById('hs-' + id);
        if (el) el.textContent = txt;
    };

    // API check
    try {
        const r = await fetch('/health');
        const d = await r.json();
        const ok = r.ok && d.status === 'ok';
        setDot('api', ok);
        setSt('api', ok ? 'Healthy' : 'Degraded');
        // If API is healthy, assume mongo+redis are connected (Go server fails to start otherwise)
        setDot('mongo', ok);
        setSt('mongo', ok ? 'Connected' : 'Unknown');
        setDot('redis', ok);
        setSt('redis', ok ? 'Connected' : 'Unknown');
    } catch {
        setDot('api', false); setSt('api', 'Unreachable');
        setDot('mongo', false); setSt('mongo', 'Unknown');
        setDot('redis', false); setSt('redis', 'Unknown');
    }

    // Hyperswitch reachability (check if payments endpoint returns auth error, not network error)
    try {
        const r = await fetch('https://sandbox.hyperswitch.io/health', { mode: 'no-cors' });
        setDot('hyperswitch', true); setSt('hyperswitch', 'Reachable');
    } catch {
        setDot('hyperswitch', false); setSt('hyperswitch', 'Check API Key / Network');
    }
}

document.getElementById('refresh-health').addEventListener('click', checkHealth);

// ---- INIT ----
updateWebhookPreview();
