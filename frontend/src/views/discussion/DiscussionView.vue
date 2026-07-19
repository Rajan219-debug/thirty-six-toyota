<template>
	<div class="discussion-view">
		<header class="discussion-header">
			<h1 class="discussion-title">
				{{ $t('navigation.discussion') }}
			</h1>
		</header>

		<div class="discussion-layout">
			<aside class="discussion-sidebar">
				<div class="discussion-sidebar__actions">
					<button
						class="discussion-btn"
						@click="openPicker(false)"
					>
						<Icon icon="plus" />
						{{ $t('discussion.memberChat') }}
					</button>
					<button
						class="discussion-btn discussion-btn--secondary"
						@click="openPicker(true)"
					>
						<Icon icon="plus" />
						{{ $t('discussion.groupChat') }}
					</button>
				</div>

				<div
					v-if="picker.open"
					class="discussion-picker"
				>
					<input
						v-if="picker.isGroup"
						v-model="picker.title"
						class="discussion-picker__title"
						:placeholder="$t('discussion.groupNamePlaceholder')"
					>
					<div class="discussion-picker__users">
						<label
							v-for="u in orgUsers"
							:key="u.id"
							class="discussion-picker__user"
						>
							<input
								:type="picker.isGroup ? 'checkbox' : 'radio'"
								name="picker-user"
								:value="u.id"
								:checked="picker.selected.includes(u.id)"
								@change="toggleSelected(u.id)"
							>
							{{ getDisplayName(u) }}
						</label>
					</div>
					<div class="discussion-picker__actions">
						<button
							class="discussion-btn discussion-btn--secondary"
							@click="picker.open = false"
						>
							{{ $t('misc.cancel') }}
						</button>
						<button
							class="discussion-btn"
							:disabled="picker.selected.length === 0"
							@click="startConversation"
						>
							{{ $t('discussion.start') }}
						</button>
					</div>
				</div>

				<div
					v-if="loadingConversations"
					class="loader-container is-loading"
				/>
				<ul
					v-else
					class="discussion-conversation-list"
				>
					<li
						v-for="c in conversations"
						:key="c.id"
					>
						<button
							class="discussion-conversation"
							:class="{'is-active': activeConversation?.id === c.id}"
							@click="selectConversation(c)"
						>
							<span class="discussion-conversation__avatar">{{ initials(conversationTitle(c)) }}</span>
							<span class="discussion-conversation__meta">
								<span class="discussion-conversation__name">
									{{ conversationTitle(c) }}
									<span
										v-if="c.isUnread"
										class="discussion-conversation__unread-dot"
									/>
								</span>
								<span class="discussion-conversation__preview">{{ c.lastMessage?.text ?? '—' }}</span>
							</span>
						</button>
					</li>
				</ul>
				<p
					v-if="!loadingConversations && conversations.length === 0"
					class="discussion-empty-hint"
				>
					{{ $t('discussion.noConversations') }}
				</p>
			</aside>

			<section class="discussion-main">
				<template v-if="activeConversation">
					<header class="discussion-thread-head">
						<span class="discussion-conversation__avatar">{{ initials(conversationTitle(activeConversation)) }}</span>
						<span class="discussion-thread-head__name">{{ conversationTitle(activeConversation) }}</span>
					</header>

					<div
						ref="threadEl"
						class="discussion-thread"
					>
						<div
							v-for="m in messages"
							:key="m.id"
							class="discussion-message"
							:class="{'is-own': m.author.id === authStore.info?.id}"
						>
							<span class="discussion-message__author">{{ getDisplayName(m.author) }}</span>
							<p class="discussion-message__text">
								{{ m.text }}
							</p>
							<span class="discussion-message__time">{{ formatTime(m.created) }}</span>
						</div>
					</div>

					<form
						class="discussion-composer"
						@submit.prevent="send"
					>
						<input
							v-model="draft"
							class="discussion-composer__input"
							:placeholder="$t('discussion.typeMessage')"
						>
						<button
							class="discussion-composer__send"
							type="submit"
							:disabled="!draft.trim()"
						>
							<Icon icon="arrow-turn-down" />
						</button>
					</form>
				</template>
				<p
					v-else
					class="discussion-empty-hint"
				>
					{{ $t('discussion.selectConversation') }}
				</p>
			</section>
		</div>
	</div>
</template>

<script setup lang="ts">
import {ref, reactive, computed, watch, nextTick, onMounted, onUnmounted} from 'vue'
import {useI18n} from 'vue-i18n'

import ConversationService from '@/services/conversation'
import ChatMessageService from '@/services/chatMessage'
import {getDisplayName} from '@/models/user'
import {useTitle} from '@/composables/useTitle'
import {useOrgUsers} from '@/composables/useOrgUsers'
import {useConversations} from '@/composables/useConversations'
import {useAuthStore} from '@/stores/auth'
import type {IConversation} from '@/modelTypes/IConversation'
import type {IChatMessage} from '@/modelTypes/IChatMessage'

const {t} = useI18n({useScope: 'global'})
useTitle(() => t('navigation.discussion'))

const authStore = useAuthStore()
const conversationService = new ConversationService()
const chatMessageService = new ChatMessageService()

const {users: allOrgUsers} = useOrgUsers()
const orgUsers = computed(() => allOrgUsers.value.filter(u => u.id !== authStore.info?.id))

const {conversations, loading: loadingConversations} = useConversations()
const activeConversation = ref<IConversation | null>(null)
const messages = ref<IChatMessage[]>([])
const draft = ref('')
const threadEl = ref<HTMLElement>()

const picker = reactive({
	open: false,
	isGroup: false,
	title: '',
	selected: [] as number[],
})

function openPicker(isGroup: boolean) {
	picker.open = true
	picker.isGroup = isGroup
	picker.title = ''
	picker.selected = []
}

function toggleSelected(id: number) {
	if (picker.isGroup) {
		const idx = picker.selected.indexOf(id)
		if (idx === -1) picker.selected.push(id)
		else picker.selected.splice(idx, 1)
	} else {
		picker.selected = [id]
	}
}

async function startConversation() {
	const conversation = await conversationService.create(picker.title, picker.isGroup, picker.selected)
	conversations.value.unshift(conversation)
	picker.open = false
	selectConversation(conversation)
}

// A 1:1 chat has no title — show the other member's name instead, matching Taskopad.
function conversationTitle(c: IConversation): string {
	if (c.isGroup) return c.title || t('discussion.groupChat')
	const other = c.members.find(m => m.id !== authStore.info?.id)
	return other ? getDisplayName(other) : (c.title || t('discussion.memberChat'))
}

function initials(name: string): string {
	return name.split(/\s+/).filter(Boolean).slice(0, 2).map(p => p[0]?.toUpperCase() ?? '').join('')
}

function formatTime(d: Date): string {
	return new Intl.DateTimeFormat(undefined, {hour: 'numeric', minute: '2-digit'}).format(d)
}

async function loadMessages(conversationId: number) {
	messages.value = await chatMessageService.getAllForConversation(conversationId)
	await nextTick()
	threadEl.value?.scrollTo(0, threadEl.value.scrollHeight)
}

function selectConversation(c: IConversation) {
	activeConversation.value = c
	// Optimistic: the message fetch this triggers marks it read server-side,
	// but that response doesn't come back through this conversation object —
	// clear it locally now so the dot disappears immediately instead of
	// waiting for the next 10s conversation-list poll.
	c.isUnread = false
}

watch(activeConversation, c => {
	if (c) loadMessages(c.id)
	else messages.value = []
})

async function send() {
	if (!draft.value.trim() || !activeConversation.value) return
	const text = draft.value.trim()
	draft.value = ''
	const message = await chatMessageService.send(activeConversation.value.id, text)
	messages.value.push(message)
	await nextTick()
	threadEl.value?.scrollTo(0, threadEl.value.scrollHeight)
}

// ponytail: no websocket/SSE infra in this app yet — polling is the lazy,
// honest way to get "near real-time" delivery. Swap for a push channel if
// message volume ever makes 4s-stale feel bad. The conversation list itself
// polls via useConversations; only the active thread's messages need their
// own faster interval here.
let messagePoll: ReturnType<typeof setInterval>

onMounted(() => {
	messagePoll = setInterval(() => {
		if (activeConversation.value) loadMessages(activeConversation.value.id)
	}, 4000)
})

onUnmounted(() => {
	clearInterval(messagePoll)
})
</script>

<style lang="scss" scoped>
.discussion-view {
	padding: 1.5rem;
	max-inline-size: 1280px;
	margin-inline: auto;
	block-size: calc(100vh - #{$navbar-height} - 3rem);
	display: flex;
	flex-direction: column;
}

.discussion-header {
	margin-block-end: 1.5rem;
}

.discussion-title {
	font-size: 1.6rem;
	font-weight: 700;
	color: var(--grey-900);
	margin: 0;
}

.discussion-layout {
	display: grid;
	grid-template-columns: 300px 1fr;
	gap: 1.25rem;
	flex: 1;
	min-block-size: 0;
}

.discussion-sidebar {
	display: flex;
	flex-direction: column;
	padding: 1rem;
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 12px;
	overflow-y: auto;

	&__actions {
		display: flex;
		gap: .5rem;
		margin-block-end: .75rem;
	}
}

.discussion-btn {
	display: inline-flex;
	align-items: center;
	gap: .4rem;
	padding: .45rem .75rem;
	border-radius: var(--control-radius);
	border: none;
	background: var(--primary);
	color: #ffffff;
	font-weight: 600;
	font-size: .8rem;
	cursor: pointer;
	flex: 1;
	justify-content: center;

	&:hover:not(:disabled) {
		opacity: .9;
	}

	&:disabled {
		opacity: .5;
		cursor: not-allowed;
	}

	&--secondary {
		background: var(--grey-100);
		color: var(--grey-700);
	}
}

.discussion-picker {
	padding: .75rem;
	margin-block-end: .75rem;
	background: var(--grey-50);
	border-radius: 8px;

	&__title {
		inline-size: 100%;
		padding: .4rem .6rem;
		margin-block-end: .5rem;
		border: 1px solid var(--card-border-color);
		border-radius: 6px;
		font-size: .85rem;
	}

	&__users {
		display: flex;
		flex-direction: column;
		gap: .3rem;
		max-block-size: 200px;
		overflow-y: auto;
		margin-block-end: .5rem;
	}

	&__user {
		display: flex;
		align-items: center;
		gap: .5rem;
		font-size: .85rem;
		color: var(--grey-700);
		cursor: pointer;
	}

	&__actions {
		display: flex;
		gap: .5rem;
	}
}

.discussion-conversation-list {
	list-style: none;
	margin: 0;
	padding: 0;
	display: flex;
	flex-direction: column;
	gap: .15rem;
	overflow-y: auto;
}

.discussion-conversation {
	display: flex;
	align-items: center;
	gap: .6rem;
	inline-size: 100%;
	padding: .5rem;
	border: none;
	background: none;
	border-radius: 8px;
	cursor: pointer;
	text-align: start;

	&:hover {
		background: var(--grey-50);
	}

	&.is-active {
		background: hsla(var(--primary-hsl), .1);
	}

	&__avatar {
		display: grid;
		place-items: center;
		flex-shrink: 0;
		inline-size: 2.25rem;
		block-size: 2.25rem;
		border-radius: 50%;
		background: var(--primary);
		color: #ffffff;
		font-size: .75rem;
		font-weight: 700;
	}

	&__meta {
		display: flex;
		flex-direction: column;
		min-inline-size: 0;
	}

	&__name {
		display: flex;
		align-items: center;
		gap: .4rem;
		font-weight: 600;
		font-size: .88rem;
		color: var(--grey-900);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	&__unread-dot {
		display: inline-block;
		flex-shrink: 0;
		inline-size: .5rem;
		block-size: .5rem;
		border-radius: 100%;
		background: var(--primary);
	}

	&__preview {
		font-size: .78rem;
		color: var(--grey-500);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
}

.discussion-empty-hint {
	color: var(--grey-400);
	font-style: italic;
	text-align: center;
	padding: 2rem 1rem;
}

.discussion-main {
	display: flex;
	flex-direction: column;
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 12px;
	overflow: hidden;
	min-block-size: 0;
}

.discussion-thread-head {
	display: flex;
	align-items: center;
	gap: .6rem;
	padding: .85rem 1.25rem;
	border-block-end: 1px solid var(--card-border-color);

	&__name {
		font-weight: 700;
		color: var(--grey-900);
	}
}

.discussion-thread {
	flex: 1;
	overflow-y: auto;
	padding: 1rem 1.25rem;
	display: flex;
	flex-direction: column;
	gap: .75rem;
}

.discussion-message {
	max-inline-size: 65%;
	padding: .5rem .75rem;
	border-radius: 10px;
	background: var(--grey-50);
	align-self: flex-start;

	&.is-own {
		background: hsla(var(--primary-hsl), .12);
		align-self: flex-end;
	}

	&__author {
		display: block;
		font-size: .7rem;
		font-weight: 700;
		color: var(--grey-600);
	}

	&__text {
		margin: .15rem 0;
		font-size: .9rem;
		color: var(--grey-900);
		white-space: pre-wrap;
	}

	&__time {
		font-size: .68rem;
		color: var(--grey-400);
	}
}

.discussion-composer {
	display: flex;
	gap: .5rem;
	padding: .85rem 1.25rem;
	border-block-start: 1px solid var(--card-border-color);

	&__input {
		flex: 1;
		padding: .55rem .85rem;
		border: 1px solid var(--card-border-color);
		border-radius: 999px;
		font-size: .9rem;
	}

	&__send {
		display: grid;
		place-items: center;
		inline-size: 2.5rem;
		block-size: 2.5rem;
		border-radius: 50%;
		border: none;
		background: var(--primary);
		color: #ffffff;
		cursor: pointer;
		flex-shrink: 0;

		&:disabled {
			opacity: .5;
			cursor: not-allowed;
		}
	}
}
</style>
