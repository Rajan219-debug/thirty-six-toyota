<template>
	<div class="documents-view">
		<header class="documents-header">
			<h1 class="documents-title">
				{{ $t('navigation.documents') }}
			</h1>
			<div class="documents-actions">
				<button
					class="documents-btn documents-btn--secondary"
					@click="creatingFolder = true"
				>
					<Icon icon="plus" />
					{{ $t('documents.newFolder') }}
				</button>
				<button
					class="documents-btn"
					:disabled="!activeFolder"
					@click="fileInput?.click()"
				>
					<Icon icon="arrow-up-from-bracket" />
					{{ $t('documents.upload') }}
				</button>
				<input
					ref="fileInput"
					type="file"
					multiple
					hidden
					@change="onFilesPicked"
				>
			</div>
		</header>

		<div class="documents-layout">
			<aside class="documents-sidebar">
				<h2 class="documents-sidebar__title">
					{{ $t('documents.myFolders') }}
				</h2>

				<form
					v-if="creatingFolder"
					class="documents-new-folder"
					@submit.prevent="submitNewFolder"
				>
					<input
						v-model="newFolderTitle"
						class="documents-new-folder__input"
						:placeholder="$t('documents.folderNamePlaceholder')"
						autofocus
					>
				</form>

				<ul class="documents-folder-list">
					<li
						v-for="folder in folders"
						:key="folder.id"
					>
						<button
							class="documents-folder"
							:class="{'is-active': activeFolder?.id === folder.id}"
							@click="activeFolder = folder"
						>
							<Icon icon="folder" />
							{{ folder.title }}
						</button>
					</li>
				</ul>

				<p
					v-if="!loadingFolders && folders.length === 0"
					class="documents-empty-hint"
				>
					{{ $t('documents.noFolders') }}
				</p>
			</aside>

			<section class="documents-main">
				<div
					v-if="loadingFolders"
					class="loader-container is-loading"
				/>
				<template v-else-if="activeFolder">
					<table
						v-if="documents.length"
						class="documents-table"
					>
						<thead>
							<tr>
								<th>{{ $t('documents.fileName') }}</th>
								<th>{{ $t('documents.owner') }}</th>
								<th>{{ $t('documents.date') }}</th>
								<th>{{ $t('documents.size') }}</th>
								<th />
							</tr>
						</thead>
						<tbody>
							<tr
								v-for="doc in documents"
								:key="doc.id"
							>
								<td class="documents-table__name">
									<Icon icon="file" />
									{{ doc.file.name }}
								</td>
								<td>{{ getDisplayName(doc.createdBy) }}</td>
								<td>{{ formatDate(doc.created) }}</td>
								<td>{{ getHumanSize(doc.file.size) }}</td>
								<td class="documents-table__actions">
									<button
										:title="$t('documents.download')"
										@click="download(doc)"
									>
										<Icon icon="download" />
									</button>
									<button
										:title="$t('misc.delete')"
										@click="remove(doc)"
									>
										<Icon icon="trash-alt" />
									</button>
								</td>
							</tr>
						</tbody>
					</table>
					<p
						v-else-if="!loadingDocuments"
						class="documents-empty-hint"
					>
						{{ $t('documents.noDocuments') }}
					</p>
				</template>
				<p
					v-else
					class="documents-empty-hint"
				>
					{{ $t('documents.selectFolder') }}
				</p>
			</section>
		</div>
	</div>
</template>

<script setup lang="ts">
import {ref, watch, onMounted} from 'vue'
import {useI18n} from 'vue-i18n'

import FolderService from '@/services/folder'
import DocumentService from '@/services/document'
import {getDisplayName} from '@/models/user'
import {getHumanSize} from '@/helpers/getHumanSize'
import {useTitle} from '@/composables/useTitle'
import type {IFolder} from '@/modelTypes/IFolder'
import type {IDocument} from '@/modelTypes/IDocument'

const {t} = useI18n({useScope: 'global'})
useTitle(() => t('navigation.documents'))

const folderService = new FolderService()
const documentService = new DocumentService()

const folders = ref<IFolder[]>([])
const activeFolder = ref<IFolder | null>(null)
const documents = ref<IDocument[]>([])
const loadingFolders = ref(true)
const loadingDocuments = ref(false)

const creatingFolder = ref(false)
const newFolderTitle = ref('')
const fileInput = ref<HTMLInputElement>()

async function loadFolders() {
	loadingFolders.value = true
	try {
		folders.value = await folderService.getAll()
		if (!activeFolder.value && folders.value.length) {
			activeFolder.value = folders.value[0]
		}
	} finally {
		loadingFolders.value = false
	}
}

async function loadDocuments(folderId: number) {
	loadingDocuments.value = true
	try {
		documents.value = await documentService.getAllForFolder(folderId)
	} finally {
		loadingDocuments.value = false
	}
}

watch(activeFolder, folder => {
	if (folder) loadDocuments(folder.id)
	else documents.value = []
})

async function submitNewFolder() {
	if (!newFolderTitle.value.trim()) return
	const folder = await folderService.create({title: newFolderTitle.value.trim()})
	folders.value.unshift(folder)
	activeFolder.value = folder
	newFolderTitle.value = ''
	creatingFolder.value = false
}

async function onFilesPicked(e: Event) {
	const input = e.target as HTMLInputElement
	if (!input.files?.length || !activeFolder.value) return
	const result = await documentService.upload(activeFolder.value.id, input.files)
	documents.value = [...result.success, ...documents.value]
	input.value = ''
}

async function download(doc: IDocument) {
	await documentService.download(doc)
}

async function remove(doc: IDocument) {
	await documentService.delete(doc)
	documents.value = documents.value.filter(d => d.id !== doc.id)
}

function formatDate(d: Date): string {
	return new Intl.DateTimeFormat(undefined, {month: 'short', day: 'numeric', year: 'numeric'}).format(d)
}

onMounted(loadFolders)
</script>

<style lang="scss" scoped>
.documents-view {
	padding: 1.5rem;
	max-inline-size: 1280px;
	margin-inline: auto;
}

.documents-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 1rem;
	margin-block-end: 1.5rem;
}

.documents-title {
	font-size: 1.6rem;
	font-weight: 700;
	color: var(--grey-900);
	margin: 0;
}

.documents-actions {
	display: flex;
	gap: .6rem;
}

.documents-btn {
	display: inline-flex;
	align-items: center;
	gap: .5rem;
	padding: .55rem 1rem;
	border-radius: var(--control-radius);
	border: none;
	background: var(--primary);
	color: #ffffff;
	font-weight: 600;
	font-size: .9rem;
	cursor: pointer;

	&:hover:not(:disabled) {
		opacity: .9;
	}

	&:disabled {
		opacity: .5;
		cursor: not-allowed;
	}

	&--secondary {
		background: var(--white);
		color: var(--grey-700);
		border: 1px solid var(--card-border-color);
	}
}

.documents-layout {
	display: grid;
	grid-template-columns: 240px 1fr;
	gap: 1.25rem;
	align-items: start;
}

.documents-sidebar {
	padding: 1rem;
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 12px;

	&__title {
		font-size: .95rem;
		font-weight: 700;
		color: var(--grey-900);
		margin: 0 0 .75rem;
	}
}

.documents-new-folder {
	margin-block-end: .75rem;

	&__input {
		inline-size: 100%;
		padding: .4rem .6rem;
		border: 1px solid var(--card-border-color);
		border-radius: 6px;
		font-size: .85rem;
	}
}

.documents-folder-list {
	list-style: none;
	margin: 0;
	padding: 0;
	display: flex;
	flex-direction: column;
	gap: .15rem;
}

.documents-folder {
	display: flex;
	align-items: center;
	gap: .6rem;
	inline-size: 100%;
	padding: .5rem .6rem;
	border: none;
	background: none;
	border-radius: 6px;
	font-size: .88rem;
	color: var(--grey-700);
	cursor: pointer;
	text-align: start;

	&:hover {
		background: var(--grey-50);
	}

	&.is-active {
		background: hsla(var(--primary-hsl), .1);
		color: var(--primary);
		font-weight: 600;
	}
}

.documents-main {
	padding: 1.25rem;
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 12px;
	min-block-size: 200px;
}

.documents-empty-hint {
	color: var(--grey-400);
	font-style: italic;
	text-align: center;
	padding: 2rem 0;
}

.documents-table {
	inline-size: 100%;
	border-collapse: collapse;

	th, td {
		padding: .65rem .75rem;
		text-align: start;
	}

	thead th {
		font-size: .7rem;
		text-transform: uppercase;
		letter-spacing: .05em;
		color: var(--grey-500);
		border-block-end: 1px solid var(--card-border-color);
	}

	tbody tr:not(:last-child) td {
		border-block-end: 1px solid var(--grey-100);
	}

	&__name {
		display: flex;
		align-items: center;
		gap: .5rem;
		font-weight: 600;
		color: var(--grey-900);
	}

	&__actions {
		display: flex;
		gap: .5rem;
		justify-content: flex-end;

		button {
			background: none;
			border: none;
			color: var(--grey-400);
			cursor: pointer;

			&:hover {
				color: var(--primary);
			}
		}
	}
}
</style>
