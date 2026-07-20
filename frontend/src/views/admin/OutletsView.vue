<template>
	<Card title="Outlets">
		<div class="outlets">
			<p v-if="loading">
				{{ $t('misc.loading') }}
			</p>
			<template v-else>
				<p class="outlets__hint">
					Each outlet (Faridabad, Palwal, …) is an isolated dealership location. Staff only see their own outlet's data; HQ users see all outlets.
				</p>

				<table class="outlets__table">
					<thead>
						<tr>
							<th>ID</th>
							<th>Name</th>
							<th>Slug</th>
							<th />
						</tr>
					</thead>
					<tbody>
						<tr
							v-for="o in outlets"
							:key="o.id"
						>
							<td>{{ o.id }}</td>
							<td>{{ o.name }}</td>
							<td><code>{{ o.slug }}</code></td>
							<td class="outlets__actions">
								<span
									v-if="o.is_default"
									class="outlets__default-tag"
								>default</span>
								<template v-else>
									<button
										type="button"
										class="outlets__make-default"
										@click="makeDefault(o)"
									>
										Make default
									</button>
									<button
										type="button"
										class="outlets__delete"
										@click="deleteOutlet(o)"
									>
										Delete
									</button>
								</template>
							</td>
						</tr>
					</tbody>
				</table>

				<h3 class="outlets__subtitle">
					Create outlet
				</h3>
				<form
					class="outlets__form"
					@submit.prevent="createOutlet"
				>
					<input
						v-model="newName"
						type="text"
						placeholder="Name (e.g. Faridabad)"
						required
					>
					<input
						v-model="newSlug"
						type="text"
						placeholder="Slug (e.g. faridabad)"
						required
					>
					<button type="submit">
						Create
					</button>
				</form>

				<h3 class="outlets__subtitle">
					Assign user to outlet
				</h3>
				<form
					class="outlets__form"
					@submit.prevent="assignUser"
				>
					<input
						v-model.number="assignUserId"
						type="number"
						min="1"
						placeholder="User ID"
						required
					>
					<select
						v-model.number="assignOutletId"
						required
					>
						<option
							v-for="o in outlets"
							:key="o.id"
							:value="o.id"
						>
							{{ o.name }}
						</option>
					</select>
					<label class="outlets__hq">
						<input
							v-model="assignIsHQ"
							type="checkbox"
						>
						HQ (sees all outlets)
					</label>
					<button type="submit">
						Assign
					</button>
				</form>
			</template>
		</div>
	</Card>
</template>

<script setup lang="ts">
import {ref, onMounted} from 'vue'
import Card from '@/components/misc/Card.vue'
import TenantService, {type ITenant} from '@/services/admin/tenantService'
import {success, error} from '@/message'

const service = new TenantService()
const outlets = ref<ITenant[]>([])
const loading = ref(false)

const newName = ref('')
const newSlug = ref('')
const assignUserId = ref<number>()
const assignOutletId = ref<number>()
const assignIsHQ = ref(false)

async function load() {
	loading.value = true
	try {
		outlets.value = await service.getAll()
	} catch (e) {
		error(e)
	} finally {
		loading.value = false
	}
}

async function createOutlet() {
	try {
		await service.create({name: newName.value, slug: newSlug.value})
		newName.value = ''
		newSlug.value = ''
		success({message: 'Outlet created.'})
		await load()
	} catch (e) {
		error(e)
	}
}

async function makeDefault(o: ITenant) {
	if (!window.confirm(`Make "${o.name}" the default outlet? New users will be created here.`)) {
		return
	}
	try {
		await service.setDefault(o.id)
		success({message: `${o.name} is now the default outlet.`})
		await load()
	} catch (e) {
		error(e)
	}
}

async function deleteOutlet(o: ITenant) {
	if (!window.confirm(`Delete outlet "${o.name}"? This can't be undone.`)) {
		return
	}
	try {
		await service.delete(o.id)
		success({message: 'Outlet deleted.'})
		await load()
	} catch (e) {
		error(e)
	}
}

async function assignUser() {
	if (!assignUserId.value || !assignOutletId.value) {
		return
	}
	try {
		await service.assignUserOutlet(assignUserId.value, assignOutletId.value, assignIsHQ.value)
		success({message: 'User assigned to outlet.'})
	} catch (e) {
		error(e)
	}
}

onMounted(load)
</script>

<style lang="scss" scoped>
.outlets__hint {
	color: var(--grey-600);
	margin-block-end: 1rem;
}

.outlets__table {
	inline-size: 100%;
	border-collapse: collapse;
	margin-block-end: 1.5rem;
}

.outlets__table th,
.outlets__table td {
	text-align: start;
	padding: 0.5rem 0.75rem;
	border-block-end: 1px solid var(--grey-200);
}

.outlets__actions {
	text-align: end;
	white-space: nowrap;
}

.outlets__make-default {
	padding: 0.3rem 0.8rem;
	margin-inline-end: 0.4rem;
	background: transparent;
	color: var(--primary);
	border: 1px solid var(--primary);
	border-radius: 4px;
	cursor: pointer;
	font-size: 0.85rem;
	transition: background 0.15s, color 0.15s;
}

.outlets__make-default:hover {
	background: var(--primary);
	color: #ffffff;
}

.outlets__delete {
	padding: 0.3rem 0.8rem;
	background: transparent;
	color: #c0392b;
	border: 1px solid #e0b4ae;
	border-radius: 4px;
	cursor: pointer;
	font-size: 0.85rem;
	transition: background 0.15s, color 0.15s, border-color 0.15s;
}

.outlets__delete:hover {
	background: #c0392b;
	color: #ffffff;
	border-color: #c0392b;
}

.outlets__default-tag {
	font-size: 0.72rem;
	color: var(--grey-500);
	text-transform: uppercase;
	letter-spacing: 0.06em;
}

.outlets__subtitle {
	font-weight: 600;
	margin-block: 1.25rem 0.5rem;
}

.outlets__form {
	display: flex;
	gap: 0.5rem;
	align-items: center;
	flex-wrap: wrap;
	margin-block-end: 1rem;
}

.outlets__form input[type='text'],
.outlets__form input[type='number'],
.outlets__form select {
	padding: 0.4rem 0.6rem;
	border: 1px solid var(--grey-300);
	border-radius: 4px;
}

.outlets__hq {
	display: inline-flex;
	align-items: center;
	gap: 0.35rem;
	color: var(--grey-700);
}

.outlets__form button {
	padding: 0.4rem 1rem;
	background: var(--primary);
	color: #ffffff;
	border: none;
	border-radius: 4px;
	cursor: pointer;
}
</style>
