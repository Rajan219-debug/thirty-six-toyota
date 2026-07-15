import AbstractService from '@/services/abstractService'
import {apiV2Url} from '@/helpers/fetcher'

export interface ITenant {
	id: number
	name: string
	slug: string
	is_default: boolean
	created: string
	updated: string
}

// Outlet (tenant) admin. All endpoints live on /api/v2/admin, so they use
// absolute v2 URLs to bypass the axios instance's /api/v1 baseURL.
export default class TenantService extends AbstractService<ITenant> {
	modelFactory(data: Partial<ITenant>) {
		return data as ITenant
	}

	async getAll(): Promise<ITenant[]> {
		const {data} = await this.http.get(apiV2Url('admin/tenants'))
		return data as ITenant[]
	}

	async create(body: {name: string, slug: string}): Promise<ITenant> {
		const {data} = await this.http.post(apiV2Url('admin/tenants'), body)
		return data as ITenant
	}

	async delete(id: number): Promise<void> {
		await this.http.delete(apiV2Url(`admin/tenants/${id}`))
	}

	async setDefault(id: number): Promise<void> {
		await this.http.patch(apiV2Url(`admin/tenants/${id}/default`))
	}

	async assignUserOutlet(userId: number, tenantId: number, isHQ: boolean) {
		const {data} = await this.http.patch(apiV2Url(`admin/users/${userId}/outlet`), {
			tenant_id: tenantId,
			is_hq: isHQ,
		})
		return data
	}
}
