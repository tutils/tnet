// DOM Elements
const tabBtns = document.querySelectorAll('.tab-btn');
const tabContents = document.querySelectorAll('.tab-content');
const addAgentBtn = document.getElementById('add-agent-btn');
const addProxyBtn = document.getElementById('add-proxy-btn');
const addAgentModal = document.getElementById('add-agent-modal');
const addProxyModal = document.getElementById('add-proxy-modal');
const addAgentForm = document.getElementById('add-agent-form');
const addProxyForm = document.getElementById('add-proxy-form');
const closeBtns = document.querySelectorAll('.close');
const agentsList = document.getElementById('agents-list');
const proxiesList = document.getElementById('proxies-list');

// Tab Switching
function switchTab(tabName) {
    // Remove active class from all tabs
    tabBtns.forEach(btn => btn.classList.remove('active'));
    tabContents.forEach(content => content.classList.remove('active'));

    // Add active class to selected tab
    document.querySelector(`[data-tab="${tabName}"]`).classList.add('active');
    document.getElementById(tabName).classList.add('active');
}

// Modal Functions
function openModal(modal) {
    modal.classList.add('show');
}

function closeModal(modal) {
    modal.classList.remove('show');
}

// Refresh Functions
function refreshAgents() {
    fetch('/api/agents')
        .then(response => response.json())
        .then(data => {
            renderServiceList(agentsList, data, 'agent');
            // Update agent count badge
            document.getElementById('agents-count').textContent = data.length;
        })
        .catch(error => {
            console.error('Error refreshing agents:', error);
        });
}

function refreshProxies() {
    fetch('/api/proxies')
        .then(response => response.json())
        .then(data => {
            renderServiceList(proxiesList, data, 'proxy');
            // Update proxy count badge
            document.getElementById('proxies-count').textContent = data.length;
        })
        .catch(error => {
            console.error('Error refreshing proxies:', error);
        });
}

// Helper function to mask crypt key values for display
function maskCryptKeyArgs(args) {
    return args.map(arg => {
        if (arg.startsWith('--crypt-key=')) {
            return '--crypt-key=**********';
        }
        return arg;
    });
}

// Copy full command with real crypt-key
function copyCommand(event) {
    const button = event.target;
    const serviceType = button.dataset.serviceType;
    const realArgs = JSON.parse(button.dataset.args);

    // Build full command: tnet agent [args] or tnet proxy [args]
    const command = `tnet ${serviceType} ${realArgs.join(' ')}`;

    // Store original state
    const originalText = button.textContent;
    const originalBackgroundColor = button.style.backgroundColor;
    const originalColor = button.style.color;

    // Copy to clipboard
    navigator.clipboard.writeText(command)
        .then(() => {
            // Subtle feedback: change button text and color temporarily
            button.textContent = 'Copied!';
            button.style.backgroundColor = '#27ae60';
            button.style.color = 'white';

            // Revert after 1.5 seconds
            setTimeout(() => {
                button.textContent = originalText;
                button.style.backgroundColor = originalBackgroundColor;
                button.style.color = originalColor;
            }, 1500);
        })
        .catch(err => {
            console.error('Failed to copy command:', err);
            // Keep alert for errors since they need user attention
            alert('Failed to copy command');
        });
}

// Render Service List
function renderServiceList(container, data, serviceType) {
    if (!data || data.length === 0) {
        container.innerHTML = '<tr><td colspan="5" class="empty-state">No ' + serviceType + 's found</td></tr>';
        return;
    }

    container.innerHTML = data.map(service => {
        // Mask crypt-key values in the args display
        const maskedArgs = maskCryptKeyArgs(service.args);
        return `
            <tr>
                <td>${service.id}</td>
                <td><span class="status-badge status-${service.status.toLowerCase()}">${service.status}</span></td>
                <td>${maskedArgs.join(' ')}</td>
                <td>
                    ${service.status === 'running' ?
                `<button class="action-btn stop-btn" onclick="stopService('${serviceType}', '${service.id}')">Stop</button>` :
                `<button class="action-btn start-btn" onclick="startService('${serviceType}', '${service.id}')">Start</button>`
            }
                    <button class="action-btn copy-command-btn" data-service-type="${serviceType}" data-args='${JSON.stringify(service.args).replace(/'/g, '&apos;')}' onclick="copyCommand(event)">Copy</button>
                    <button class="action-btn delete-btn" onclick="deleteService('${serviceType}', '${service.id}')">Delete</button>
                </td>
            </tr>
        `;
    }).join('');
}

// Service Actions
function startService(serviceType, id) {
    if (serviceType === 'agent') {
        // For existing agents, always use restart endpoint
        restartAgent(id);
    } else if (serviceType === 'proxy') {
        // For existing proxies, always use restart endpoint
        restartProxy(id);
    }
}

function stopService(serviceType, id) {
    if (serviceType === 'agent') {
        stopAgent(id);
    } else if (serviceType === 'proxy') {
        stopProxy(id);
    }
}

function deleteService(serviceType, id) {
    if (confirm(`Are you sure you want to delete this ${serviceType}?`)) {
        if (serviceType === 'agent') {
            deleteAgent(id);
        } else if (serviceType === 'proxy') {
            deleteProxy(id);
        }
    }
}

// Agent Functions
function startAgent(args) {
    fetch('/api/agents/start', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ args: args })
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                refreshAgents();
            } else {
                alert('Failed to start agent: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error starting agent:', error);
            alert('Failed to start agent');
        });
}

function restartAgent(id) {
    fetch(`/api/agents/restart?id=${id}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        }
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                refreshAgents();
            } else {
                alert('Failed to restart agent: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error restarting agent:', error);
            alert('Failed to restart agent');
        });
}

function stopAgent(id) {
    fetch(`/api/agents/stop?id=${id}`, {
        method: 'POST'
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                refreshAgents();
            } else {
                alert('Failed to stop agent: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error stopping agent:', error);
            alert('Failed to stop agent');
        });
}

function deleteAgent(id) {
    fetch(`/api/agents/delete?id=${id}`, {
        method: 'POST'
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                refreshAgents();
            } else {
                alert('Failed to delete agent: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error deleting agent:', error);
            alert('Failed to delete agent');
        });
}

// Proxy Functions
function startProxy(args) {
    fetch('/api/proxies/start', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ args: args })
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                refreshProxies();
            } else {
                alert('Failed to start proxy: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error starting proxy:', error);
            alert('Failed to start proxy');
        });
}

function restartProxy(id) {
    fetch(`/api/proxies/restart?id=${id}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        }
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                refreshProxies();
            } else {
                alert('Failed to restart proxy: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error restarting proxy:', error);
            alert('Failed to restart proxy');
        });
}

function stopProxy(id) {
    fetch(`/api/proxies/stop?id=${id}`, {
        method: 'POST'
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                refreshProxies();
            } else {
                alert('Failed to stop proxy: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error stopping proxy:', error);
            alert('Failed to stop proxy');
        });
}

function deleteProxy(id) {
    fetch(`/api/proxies/delete?id=${id}`, {
        method: 'POST'
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                refreshProxies();
            } else {
                alert('Failed to delete proxy: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error deleting proxy:', error);
            alert('Failed to delete proxy');
        });
}

// Crypt Key Functions
function toggleCryptKeyVisibility(serviceType) {
    const input = document.getElementById(`${serviceType}-crypt-key`);
    const container = input.parentElement;
    const button = container.querySelector('.toggle-visibility');

    if (input.type === 'password') {
        input.type = 'text';
        button.textContent = '🙈';
    } else {
        input.type = 'password';
        button.textContent = '👁️';
    }
}

function copyCryptKey(serviceType) {
    const input = document.getElementById(`${serviceType}-crypt-key`);
    const originalType = input.type;

    // Change to text temporarily to copy value
    input.type = 'text';
    input.select();
    input.setSelectionRange(0, 99999);

    try {
        document.execCommand('copy');
        // Show feedback with color change only
        const container = input.parentElement;
        const buttons = container.querySelectorAll('.input-buttons button');
        let copyBtn;
        buttons.forEach(btn => {
            if (btn.title === 'Copy') {
                copyBtn = btn;
            }
        });

        if (copyBtn) {
            const originalBg = copyBtn.style.backgroundColor;
            copyBtn.style.backgroundColor = '#27ae60';

            setTimeout(() => {
                copyBtn.style.backgroundColor = originalBg;
            }, 1000);
        }
    } catch (err) {
        console.error('Failed to copy:', err);
        alert('Failed to copy crypt key');
    } finally {
        // Restore original type
        input.type = originalType;
    }
}

function generateRandomCryptKey(serviceType) {
    // Generate a random 13-digit number
    const randomKey = Math.floor(1000000000000 + Math.random() * 9000000000000).toString();
    const input = document.getElementById(`${serviceType}-crypt-key`);
    input.value = randomKey;
}

// Event Listeners
// Tab buttons
window.addEventListener('DOMContentLoaded', () => {
    tabBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            switchTab(btn.getAttribute('data-tab'));
        });
    });

    // Add buttons
    addAgentBtn.addEventListener('click', () => {
        openModal(addAgentModal);
    });

    addProxyBtn.addEventListener('click', () => {
        openModal(addProxyModal);
    });

    // Close buttons
    closeBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            closeModal(btn.closest('.modal'));
        });
    });

    // Close modal when clicking outside
    window.addEventListener('click', (event) => {
        if (event.target.classList.contains('modal')) {
            closeModal(event.target);
        }
    });

    // Helper function to get form group containing an element
    function getFormGroup(element) {
        let parent = element.parentElement;
        while (parent && !parent.classList.contains('form-group')) {
            parent = parent.parentElement;
        }
        return parent;
    }

    // Helper function to parse command line arguments
    function parseCommandArgs(cmdLine) {
        // Basic argument parsing that handles spaces in quotes
        const args = [];
        let currentArg = '';
        let inQuotes = false;
        let escaped = false;

        for (let i = 0; i < cmdLine.length; i++) {
            const char = cmdLine[i];

            if (escaped) {
                currentArg += char;
                escaped = false;
            } else if (char === '\\') {
                escaped = true;
            } else if (char === '"') {
                inQuotes = !inQuotes;
            } else if (char === ' ' && !inQuotes) {
                if (currentArg) {
                    args.push(currentArg);
                    currentArg = '';
                }
            } else {
                currentArg += char;
            }
        }

        if (currentArg) {
            args.push(currentArg);
        }

        return args;
    }

    // Agent form: tunnel mode radio buttons
    const agentTunnelListenRadio = document.querySelector('input[name="agent-tunnel-mode"][value="listen"]');
    const agentTunnelConnectRadio = document.querySelector('input[name="agent-tunnel-mode"][value="connect"]');
    const agentTunnelListenGroup = getFormGroup(document.getElementById('agent-tunnel-listen'));
    const agentTunnelConnectGroup = getFormGroup(document.getElementById('agent-tunnel-connect'));

    // Set initial visibility for agent tunnel mode
    if (agentTunnelListenRadio.checked) {
        agentTunnelListenGroup.classList.remove('hidden');
        agentTunnelConnectGroup.classList.add('hidden');
    } else {
        agentTunnelListenGroup.classList.add('hidden');
        agentTunnelConnectGroup.classList.remove('hidden');
    }

    agentTunnelListenRadio.addEventListener('change', () => {
        agentTunnelListenGroup.classList.remove('hidden');
        agentTunnelConnectGroup.classList.add('hidden');
    });

    agentTunnelConnectRadio.addEventListener('change', () => {
        agentTunnelListenGroup.classList.add('hidden');
        agentTunnelConnectGroup.classList.remove('hidden');
    });

    // Proxy form: tunnel mode radio buttons
    const proxyTunnelListenRadio = document.querySelector('input[name="proxy-tunnel-mode"][value="listen"]');
    const proxyTunnelConnectRadio = document.querySelector('input[name="proxy-tunnel-mode"][value="connect"]');
    const proxyTunnelListenGroup = getFormGroup(document.getElementById('proxy-tunnel-listen'));
    const proxyTunnelConnectGroup = getFormGroup(document.getElementById('proxy-tunnel-connect'));

    // Set initial visibility for proxy tunnel mode
    if (proxyTunnelListenRadio.checked) {
        proxyTunnelListenGroup.classList.remove('hidden');
        proxyTunnelConnectGroup.classList.add('hidden');
    } else {
        proxyTunnelListenGroup.classList.add('hidden');
        proxyTunnelConnectGroup.classList.remove('hidden');
    }

    proxyTunnelListenRadio.addEventListener('change', () => {
        proxyTunnelListenGroup.classList.remove('hidden');
        proxyTunnelConnectGroup.classList.add('hidden');
    });

    proxyTunnelConnectRadio.addEventListener('change', () => {
        proxyTunnelListenGroup.classList.add('hidden');
        proxyTunnelConnectGroup.classList.remove('hidden');
    });

    // Proxy form: mode radio buttons
    const proxyModeRadio = document.querySelector('input[name="proxy-mode"][value="proxy"]');
    const proxyExecuteRadio = document.querySelector('input[name="proxy-mode"][value="execute"]');
    const proxyListenGroup = getFormGroup(document.getElementById('proxy-listen'));
    const proxyConnectGroup = getFormGroup(document.getElementById('proxy-connect'));
    const proxyExecuteGroup = getFormGroup(document.getElementById('proxy-execute'));
    const proxyRawPtyGroup = getFormGroup(document.getElementById('proxy-raw-pty'));

    proxyModeRadio.addEventListener('change', () => {
        proxyListenGroup.classList.remove('hidden');
        proxyConnectGroup.classList.remove('hidden');
        proxyExecuteGroup.classList.add('hidden');
        proxyRawPtyGroup.classList.add('hidden');
    });

    proxyExecuteRadio.addEventListener('change', () => {
        proxyListenGroup.classList.add('hidden');
        proxyConnectGroup.classList.add('hidden');
        proxyExecuteGroup.classList.remove('hidden');
        proxyRawPtyGroup.classList.remove('hidden');
    });

    // Agent form: input mode toggle
    const agentFormModeRadio = document.querySelector('input[name="agent-input-mode"][value="form"]');
    const agentDirectModeRadio = document.querySelector('input[name="agent-input-mode"][value="direct"]');
    const agentFormFields = document.getElementById('agent-form-fields');
    const agentDirectFields = document.getElementById('agent-direct-fields');

    agentFormModeRadio.addEventListener('change', () => {
        agentFormFields.classList.remove('hidden');
        agentDirectFields.classList.add('hidden');
    });

    agentDirectModeRadio.addEventListener('change', () => {
        agentFormFields.classList.add('hidden');
        agentDirectFields.classList.remove('hidden');
    });

    // Proxy form: input mode toggle
    const proxyFormModeRadio = document.querySelector('input[name="proxy-input-mode"][value="form"]');
    const proxyDirectModeRadio = document.querySelector('input[name="proxy-input-mode"][value="direct"]');
    const proxyFormFields = document.getElementById('proxy-form-fields');
    const proxyDirectFields = document.getElementById('proxy-direct-fields');

    proxyFormModeRadio.addEventListener('change', () => {
        proxyFormFields.classList.remove('hidden');
        proxyDirectFields.classList.add('hidden');
    });

    proxyDirectModeRadio.addEventListener('change', () => {
        proxyFormFields.classList.add('hidden');
        proxyDirectFields.classList.remove('hidden');
    });

    // Form submissions
    addAgentForm.addEventListener('submit', (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);
        let args = [];

        // Check input mode
        const inputMode = formData.get('agent-input-mode');

        if (inputMode === 'direct') {
            // Direct command mode - only use direct command parameters
            const directCommand = formData.get('direct-command');
            if (directCommand) {
                args = parseCommandArgs(directCommand);
                // Remove command name if present (agent/proxy), as server will add it automatically
                if (args.length > 0 && (args[0] === 'agent' || args[0] === 'proxy')) {
                    args = args.slice(1);
                }
            }
        } else {
            // Form mode
            // Tunnel mode
            const tunnelMode = formData.get('agent-tunnel-mode');
            if (tunnelMode === 'listen') {
                const tunnelListen = formData.get('tunnel-listen');
                if (tunnelListen) args.push('--tunnel-listen=' + tunnelListen);
            } else {
                const tunnelConnect = formData.get('tunnel-connect');
                if (tunnelConnect) args.push('--tunnel-connect=' + tunnelConnect);
            }

            // Crypt key
            const cryptKey = formData.get('crypt-key');
            if (cryptKey) args.push('--crypt-key=' + cryptKey);

            // Enabled execute
            if (formData.has('enabled-execute')) {
                args.push('--enabled-execute');
            }
        }

        // Show creating state
        const submitBtn = addAgentForm.querySelector('.submit-btn');
        const originalText = submitBtn.textContent;
        submitBtn.textContent = 'Creating...';
        submitBtn.disabled = true;

        fetch('/api/agents/start', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                args: args
            })
        })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    closeModal(addAgentModal);
                    // Reset form and ensure correct UI state
                    addAgentForm.reset();
                    // Explicitly show form fields and hide direct command fields
                    const agentFormFields = document.getElementById('agent-form-fields');
                    const agentDirectFields = document.getElementById('agent-direct-fields');
                    agentFormFields.classList.remove('hidden');
                    agentDirectFields.classList.add('hidden');
                    // Wait 2 seconds before refreshing to show creating state
                    setTimeout(() => {
                        refreshAgents();
                    }, 2000);
                } else {
                    alert('Failed to add agent: ' + data.error);
                }
            })
            .catch(error => {
                console.error('Error adding agent:', error);
                alert('Failed to add agent');
            })
            .finally(() => {
                // Restore submit button
                submitBtn.textContent = originalText;
                submitBtn.disabled = false;
            });
    });

    addProxyForm.addEventListener('submit', (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);
        let args = [];

        // Check input mode
        const inputMode = formData.get('proxy-input-mode');

        if (inputMode === 'direct') {
            // Direct command mode - only use direct command parameters
            const directCommand = formData.get('direct-command');
            if (directCommand) {
                args = parseCommandArgs(directCommand);
                // Remove command name if present (agent/proxy), as server will add it automatically
                if (args.length > 0 && (args[0] === 'agent' || args[0] === 'proxy')) {
                    args = args.slice(1);
                }
            }
        } else {
            // Form mode
            // Tunnel mode
            const tunnelMode = formData.get('proxy-tunnel-mode');
            if (tunnelMode === 'listen') {
                const tunnelListen = formData.get('tunnel-listen');
                if (tunnelListen) args.push('--tunnel-listen=' + tunnelListen);
            } else {
                const tunnelConnect = formData.get('tunnel-connect');
                if (tunnelConnect) args.push('--tunnel-connect=' + tunnelConnect);
            }

            // Mode
            const mode = formData.get('proxy-mode');
            if (mode === 'proxy') {
                const listen = formData.get('listen');
                if (listen) args.push('--listen=' + listen);

                const connect = formData.get('connect');
                if (connect) args.push('--connect=' + connect);
            } else {
                const execute = formData.get('execute');
                if (execute) args.push('--execute=' + execute);

                if (formData.has('raw-pty')) {
                    args.push('--raw-pty');
                }
            }

            // Crypt key
            const cryptKey = formData.get('crypt-key');
            if (cryptKey) args.push('--crypt-key=' + cryptKey);

            // Dump dir
            const dumpDir = formData.get('dump-dir');
            if (dumpDir) args.push('--dump-dir=' + dumpDir);
        }

        // Show creating state
        const submitBtn = addProxyForm.querySelector('.submit-btn');
        const originalText = submitBtn.textContent;
        submitBtn.textContent = 'Creating...';
        submitBtn.disabled = true;

        fetch('/api/proxies/start', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                args: args
            })
        })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    closeModal(addProxyModal);
                    // Reset form and ensure correct UI state
                    addProxyForm.reset();
                    // Explicitly show form fields and hide direct command fields
                    const proxyFormFields = document.getElementById('proxy-form-fields');
                    const proxyDirectFields = document.getElementById('proxy-direct-fields');
                    proxyFormFields.classList.remove('hidden');
                    proxyDirectFields.classList.add('hidden');
                    // Wait 2 seconds before refreshing to show creating state
                    setTimeout(() => {
                        refreshProxies();
                    }, 2000);
                } else {
                    alert('Failed to add proxy: ' + data.error);
                }
            })
            .catch(error => {
                console.error('Error adding proxy:', error);
                alert('Failed to add proxy');
            })
            .finally(() => {
                // Restore submit button
                submitBtn.textContent = originalText;
                submitBtn.disabled = false;
            });
    });

    // Initial refresh
    refreshAgents();
    refreshProxies();

    // Single polling mechanism: refresh active tab every 10 seconds
    // This reduces API calls while keeping the interface responsive
    setInterval(() => {
        if (document.querySelector('#agents.active')) {
            refreshAgents();
        } else if (document.querySelector('#proxies.active')) {
            refreshProxies();
        }
    }, 10000);
});

