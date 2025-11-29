/**
 * API Keys Store - manages API keys state with Svelte 5 runes
 */
import type { APIKey, Tenant } from '$lib/api/apikeys';

type TabType = 'personal' | 'tenant';

class APIKeysStore {
	// State
	personalKeys = $state<APIKey[]>([]);
	tenantKeys = $state<APIKey[]>([]);
	tenants = $state<Tenant[]>([]);
	currentTenant = $state<Tenant | null>(null);
	activeTab = $state<TabType>('personal');
	isLoading = $state(false);
	
	// For showing the created key
	createdKeyValue = $state<string | null>(null);

	// Derived
	currentKeys = $derived(
		this.activeTab === 'personal' ? this.personalKeys : this.tenantKeys
	);

	hasPersonalKeys = $derived(this.personalKeys.length > 0);
	hasTenantKeys = $derived(this.tenantKeys.length > 0);
	hasTenants = $derived(this.tenants.length > 0);

	// Actions
	setPersonalKeys(keys: APIKey[]) {
		this.personalKeys = keys;
	}

	setTenantKeys(keys: APIKey[]) {
		this.tenantKeys = keys;
	}

	setTenants(tenants: Tenant[]) {
		this.tenants = tenants;
	}

	setCurrentTenant(tenant: Tenant | null) {
		this.currentTenant = tenant;
	}

	setActiveTab(tab: TabType) {
		this.activeTab = tab;
	}

	setLoading(loading: boolean) {
		this.isLoading = loading;
	}

	setCreatedKeyValue(key: string | null) {
		this.createdKeyValue = key;
	}

	// Add new key to appropriate list
	addPersonalKey(key: APIKey) {
		this.personalKeys = [key, ...this.personalKeys];
	}

	addTenantKey(key: APIKey) {
		this.tenantKeys = [key, ...this.tenantKeys];
	}

	// Remove key from list
	removePersonalKey(keyId: string) {
		this.personalKeys = this.personalKeys.filter((k) => k.id !== keyId);
	}

	removeTenantKey(keyId: string) {
		this.tenantKeys = this.tenantKeys.filter((k) => k.id !== keyId);
	}

	// Clear state
	clear() {
		this.personalKeys = [];
		this.tenantKeys = [];
		this.currentTenant = null;
		this.createdKeyValue = null;
	}
}

export const apiKeysStore = new APIKeysStore();

