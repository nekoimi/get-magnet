export interface PageQuery {
	page_num: number;
	page_size: number;
}

export interface CrawlerWorker {
	id: number;
	running: boolean;
	current_url?: string;
}

export interface CrawlerStatus {
	worker_count: number;
	running: number;
	queue_length: number;
	workers: CrawlerWorker[];
}

export interface CrawlerProvider {
	name: string;
	cron: string;
	enabled: boolean;
}

export interface JobSnapshot {
	name: string;
	cron: string;
	next_run?: string;
	last_run?: string;
	status?: 'success' | 'failed';
	duration_ms?: number;
	error?: string;
}

export interface ServiceHealth {
	ok: boolean;
	message?: string;
	latency_ms: number;
}

export interface VersionInfo {
	version?: string;
	commit?: string;
	go_version?: string;
	started_at?: string;
	uptime_seconds?: number;
}

export function formatDateTime(value?: string | number | Date | null): string {
	if (value === undefined || value === null || value === '') return '-';
	const date = value instanceof Date ? value : new Date(value);
	if (Number.isNaN(date.getTime())) return String(value);
	return new Intl.DateTimeFormat('zh-CN', {
		year: 'numeric',
		month: '2-digit',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit',
		second: '2-digit',
		hour12: false,
	}).format(date);
}

export function getErrorMessage(error: unknown, fallback = '操作失败'): string {
	if (typeof error === 'string' && error) return error;
	if (error && typeof error === 'object') {
		const value = error as { msg?: unknown; message?: unknown };
		if (typeof value.msg === 'string' && value.msg) return value.msg;
		if (typeof value.message === 'string' && value.message) return value.message;
	}
	return fallback;
}
