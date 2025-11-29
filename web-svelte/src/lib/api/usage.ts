import { api } from './client';

export interface UsageStats {
	period: string;
	total_requests: number;
	total_tokens: number;
	total_input_tokens: number;
	total_output_tokens: number;
	by_model: Array<{
		model: string;
		requests: number;
		tokens: number;
	}>;
	by_day: Array<{
		date: string;
		requests: number;
		tokens: number;
	}>;
}

export interface UsageRecord {
	id: string;
	timestamp: string;
	model: string;
	input_tokens: number;
	output_tokens: number;
	total_tokens: number;
	latency_ms: number;
	conversation_id?: string;
}

export interface UsageResponse {
	stats: UsageStats;
	records: UsageRecord[];
	total: number;
}

export const usageApi = {
	getUsage: (period: '7d' | '30d' | '90d' = '30d', page = 1, perPage = 50) => {
		const params = new URLSearchParams({
			period,
			page: String(page),
			per_page: String(perPage)
		});
		return api.get<UsageResponse>(`/api/usage?${params}`);
	},

	getStats: (period: '7d' | '30d' | '90d' = '30d') => {
		return api.get<UsageStats>(`/api/usage/stats?period=${period}`);
	}
};

