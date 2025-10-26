// Admin Audit Log Management
// Version 1.11.4+: Enhanced Audit Logging

document.addEventListener('DOMContentLoaded', function() {
    console.log('Admin Audit Log page loaded');
    
    // Initialize
    loadAuditStats();
    loadAuditEvents();

    // Event listeners
    document.getElementById('apply-filters-btn')?.addEventListener('click', applyFilters);
    document.getElementById('reset-filters-btn')?.addEventListener('click', resetFilters);
    document.getElementById('export-csv-btn')?.addEventListener('click', exportToCSV);
    document.getElementById('prev-page-btn')?.addEventListener('click', prevPage);
    document.getElementById('next-page-btn')?.addEventListener('click', nextPage);
});

let currentPage = 1;
const pageSize = 50;

// Load audit statistics
async function loadAuditStats() {
    try {
        const response = await fetch('/api/admin/audit/stats', {
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('access_token')}`
            }
        });

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const stats = await response.json();
        
        // Update stats cards
        document.getElementById('critical-events').textContent = stats.by_severity?.critical || 0;
        document.getElementById('warning-events').textContent = stats.by_severity?.warning || 0;
        document.getElementById('info-events').textContent = stats.by_severity?.info || 0;
        document.getElementById('failed-logins').textContent = stats.failed_logins_24h || 0;

    } catch (error) {
        console.error('Failed to load stats:', error);
        showError('Failed to load audit statistics: ' + error.message);
    }
}

// Load audit events
async function loadAuditEvents() {
    const filters = getFilters();
    
    try {
        const queryParams = new URLSearchParams({
            page: currentPage,
            page_size: pageSize,
            ...filters
        });

        const response = await fetch(`/api/admin/audit?${queryParams}`, {
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('access_token')}`
            }
        });

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const data = await response.json();
        
        // Display events
        displayAuditEvents(data.events || []);
        
        // Update pagination
        updatePagination(data.total || 0);

    } catch (error) {
        console.error('Failed to load audit events:', error);
        showError('Failed to load audit events: ' + error.message);
        document.querySelector('#audit-events-table tbody').innerHTML = `
            <tr><td colspan="8" class="text-center text-danger">Error: ${error.message}</td></tr>
        `;
    }
}

// Get filters from form
function getFilters() {
    const filters = {};
    
    const eventType = document.getElementById('filter-event-type')?.value;
    if (eventType && eventType !== 'all') {
        filters.event_type = eventType;
    }
    
    const severity = document.getElementById('filter-severity')?.value;
    if (severity && severity !== 'all') {
        filters.severity = severity;
    }
    
    const resource = document.getElementById('filter-resource')?.value;
    if (resource && resource !== 'all') {
        filters.resource_type = resource;
    }
    
    const status = document.getElementById('filter-status')?.value;
    if (status && status !== 'all') {
        filters.status = status;
    }
    
    const fromDate = document.getElementById('filter-from-date')?.value;
    if (fromDate) {
        filters.from_date = new Date(fromDate).toISOString();
    }
    
    const toDate = document.getElementById('filter-to-date')?.value;
    if (toDate) {
        filters.to_date = new Date(toDate).toISOString();
    }
    
    const actorID = document.getElementById('filter-actor-id')?.value;
    if (actorID) {
        filters.actor_id = actorID;
    }
    
    return filters;
}

// Display audit events in table
function displayAuditEvents(events) {
    const tbody = document.querySelector('#audit-events-table tbody');
    
    if (!events || events.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" class="text-center">No audit events found</td></tr>';
        return;
    }
    
    tbody.innerHTML = events.map(event => `
        <tr class="severity-${event.severity}">
            <td>${formatTimestamp(event.timestamp)}</td>
            <td><span class="badge bg-secondary">${event.event_type}</span></td>
            <td><span class="badge bg-${getSeverityColor(event.severity)}">${event.severity}</span></td>
            <td>${formatActor(event.actor_id, event.actor_username)}</td>
            <td>${event.action}</td>
            <td>${event.resource_type || '-'}</td>
            <td><span class="badge bg-${getStatusColor(event.status)}">${event.status}</span></td>
            <td>${event.ip_address || '-'}</td>
        </tr>
    `).join('');
}

// Format timestamp
function formatTimestamp(timestamp) {
    const date = new Date(timestamp);
    return date.toLocaleString('en-GB', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
    });
}

// Format actor
function formatActor(actorID, actorUsername) {
    if (actorUsername) {
        return `<strong>${actorUsername}</strong><br/><small class="text-muted">${actorID}</small>`;
    }
    return actorID || '-';
}

// Get severity badge color
function getSeverityColor(severity) {
    switch (severity) {
        case 'critical': return 'danger';
        case 'warning': return 'warning';
        case 'info': return 'info';
        default: return 'secondary';
    }
}

// Get status badge color
function getStatusColor(status) {
    switch (status) {
        case 'success': return 'success';
        case 'failure': return 'danger';
        case 'pending': return 'warning';
        default: return 'secondary';
    }
}

// Update pagination
function updatePagination(total) {
    const totalPages = Math.ceil(total / pageSize);
    document.getElementById('page-info').textContent = `Page ${currentPage} of ${totalPages}`;
    
    document.getElementById('prev-page-btn').disabled = currentPage === 1;
    document.getElementById('next-page-btn').disabled = currentPage >= totalPages;
    
    document.getElementById('total-events').textContent = `Showing ${Math.min((currentPage - 1) * pageSize + 1, total)}-${Math.min(currentPage * pageSize, total)} of ${total} events`;
}

// Apply filters
function applyFilters() {
    currentPage = 1;
    loadAuditEvents();
}

// Reset filters
function resetFilters() {
    document.getElementById('filter-event-type').value = 'all';
    document.getElementById('filter-severity').value = 'all';
    document.getElementById('filter-resource').value = 'all';
    document.getElementById('filter-status').value = 'all';
    document.getElementById('filter-from-date').value = '';
    document.getElementById('filter-to-date').value = '';
    document.getElementById('filter-actor-id').value = '';
    
    currentPage = 1;
    loadAuditEvents();
}

// Export to CSV
async function exportToCSV() {
    const filters = getFilters();
    
    try {
        const queryParams = new URLSearchParams(filters);
        
        const response = await fetch(`/api/admin/audit/export?${queryParams}`, {
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('access_token')}`
            }
        });

        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }

        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `audit-log-${new Date().toISOString().split('T')[0]}.csv`;
        document.body.appendChild(a);
        a.click();
        window.URL.revokeObjectURL(url);
        document.body.removeChild(a);
        
        showSuccess('Audit log exported successfully');

    } catch (error) {
        console.error('Failed to export audit log:', error);
        showError('Failed to export audit log: ' + error.message);
    }
}

// Pagination
function prevPage() {
    if (currentPage > 1) {
        currentPage--;
        loadAuditEvents();
    }
}

function nextPage() {
    currentPage++;
    loadAuditEvents();
}

// Show success message
function showSuccess(message) {
    // TODO: Use proper notification system
    alert(message);
}

// Show error message
function showError(message) {
    // TODO: Use proper notification system
    console.error(message);
}

