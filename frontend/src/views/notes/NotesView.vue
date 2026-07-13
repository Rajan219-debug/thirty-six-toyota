<template>
	<div class="notes-view">
		<header class="notes-header">
			<h1 class="notes-title">
				{{ $t('navigation.notes') }}
			</h1>
			<button
				class="notes-add"
				@click="startNew"
			>
				<Icon icon="plus" />
				{{ $t('notes.add') }}
			</button>
		</header>

		<div
			v-if="editing"
			class="note-form"
		>
			<input
				v-model="draftTitle"
				class="note-form__title"
				:placeholder="$t('notes.titlePlaceholder')"
				autofocus
				@keydown.enter="save"
			>
			<textarea
				v-model="draftBody"
				class="note-form__body"
				rows="4"
			/>
			<div class="note-form__actions">
				<XButton
					variant="secondary"
					@click="editing = false"
				>
					{{ $t('misc.cancel') }}
				</XButton>
				<XButton
					:disabled="!draftTitle.trim()"
					@click="save"
				>
					{{ $t('misc.save') }}
				</XButton>
			</div>
		</div>

		<p
			v-if="notes.length === 0 && !editing"
			class="notes-empty"
		>
			{{ $t('notes.empty') }}
		</p>

		<div
			v-else
			class="notes-grid"
		>
			<article
				v-for="note in sortedNotes"
				:key="note.id"
				class="note-card"
				:class="`note-card--${note.color}`"
			>
				<div class="note-card__head">
					<h2 class="note-card__title">
						{{ note.title }}
					</h2>
					<button
						class="note-card__pin"
						:class="{'is-pinned': note.pinned}"
						@click="togglePin(note.id)"
					>
						<Icon icon="star" />
					</button>
				</div>
				<p class="note-card__body">
					{{ note.body }}
				</p>
				<button
					class="note-card__delete"
					@click="remove(note.id)"
				>
					<Icon icon="trash-alt" />
				</button>
			</article>
		</div>
	</div>
</template>

<script setup lang="ts">
import {ref, computed} from 'vue'
import {useI18n} from 'vue-i18n'
import {storeToRefs} from 'pinia'

import XButton from '@/components/input/Button.vue'
import {useNotesStore} from '@/stores/notes'
import {useTitle} from '@/composables/useTitle'

const {t} = useI18n({useScope: 'global'})
useTitle(() => t('navigation.notes'))

const notesStore = useNotesStore()
const {notes} = storeToRefs(notesStore)
const {add, remove, togglePin} = notesStore

const sortedNotes = computed(() => [...notes.value].sort((a, b) => {
	if (a.pinned !== b.pinned) return a.pinned ? -1 : 1
	return b.updated - a.updated
}))

const editing = ref(false)
const draftTitle = ref('')
const draftBody = ref('')

function startNew() {
	draftTitle.value = ''
	draftBody.value = ''
	editing.value = true
}

function save() {
	if (!draftTitle.value.trim()) return
	add(draftTitle.value.trim(), draftBody.value.trim())
	editing.value = false
}
</script>

<style lang="scss" scoped>
.notes-view {
	padding: 1.5rem;
	max-inline-size: 1100px;
	margin-inline: auto;
}

.notes-header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 1rem;
	margin-block-end: 1.5rem;
}

.notes-title {
	font-size: 1.6rem;
	font-weight: 700;
	color: var(--grey-900);
	margin: 0;
}

.notes-add {
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

	&:hover {
		opacity: .9;
	}
}

.note-form {
	display: flex;
	flex-direction: column;
	gap: .6rem;
	padding: 1rem;
	margin-block-end: 1.5rem;
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 12px;

	&__title {
		border: none;
		border-block-end: 1px solid var(--card-border-color);
		padding-block-end: .4rem;
		font-size: 1.1rem;
		font-weight: 600;

		&:focus {
			outline: none;
		}
	}

	&__body {
		border: none;
		resize: vertical;
		font-family: inherit;

		&:focus {
			outline: none;
		}
	}

	&__actions {
		display: flex;
		justify-content: flex-end;
		gap: .5rem;
	}
}

.notes-empty {
	color: var(--grey-400);
	font-style: italic;
}

.notes-grid {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
	gap: 1rem;
}

.note-card {
	position: relative;
	padding: 1.25rem;
	border-radius: 12px;
	min-block-size: 140px;

	&--amber {
		background: hsl(42, 90%, 88%);
	}

	&--mint {
		background: hsl(174, 70%, 90%);
	}

	&--lavender {
		background: hsl(268, 60%, 92%);
	}

	&__head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: .5rem;
	}

	&__title {
		font-size: 1.05rem;
		font-weight: 700;
		color: var(--grey-900);
		margin: 0;
	}

	&__pin {
		background: none;
		border: none;
		cursor: pointer;
		color: var(--grey-400);
		flex-shrink: 0;

		&.is-pinned {
			color: var(--warning);
		}
	}

	&__body {
		margin-block-start: .5rem;
		color: var(--grey-700);
		white-space: pre-wrap;
	}

	&__delete {
		position: absolute;
		inset-block-end: .75rem;
		inset-inline-end: .75rem;
		background: none;
		border: none;
		cursor: pointer;
		color: var(--grey-400);
		opacity: 0;
		transition: opacity .15s ease;

		&:hover {
			color: var(--danger);
		}
	}

	&:hover &__delete {
		opacity: 1;
	}
}
</style>
