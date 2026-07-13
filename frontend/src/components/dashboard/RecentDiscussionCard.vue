<template>
	<section class="recent-discussion">
		<div class="recent-discussion__head">
			<h2>{{ $t('dashboard.discussion.title') }}</h2>
			<RouterLink
				:to="{name: 'discussion.index'}"
				class="recent-discussion__link"
			>
				{{ $t('dashboard.viewAll') }} <Icon icon="angle-right" />
			</RouterLink>
		</div>

		<p
			v-if="loading"
			class="recent-discussion__empty"
		>
			{{ $t('misc.loading') }}
		</p>
		<p
			v-else-if="conversations.length === 0"
			class="recent-discussion__empty"
		>
			{{ $t('dashboard.discussion.empty') }}
		</p>
		<ul
			v-else
			class="recent-discussion__list"
		>
			<li
				v-for="c in conversations"
				:key="c.id"
				class="recent-discussion__item"
				@click="goToConversation()"
			>
				<span class="recent-discussion__avatar">{{ initials(conversationTitle(c)) }}</span>
				<div class="recent-discussion__meta">
					<div class="recent-discussion__top">
						<span class="recent-discussion__name">{{ conversationTitle(c) }}</span>
						<span
							v-if="c.lastMessage"
							class="recent-discussion__date"
						>{{ formatDateSince(c.lastMessage.created) }}</span>
					</div>
					<p class="recent-discussion__snippet">
						{{ c.lastMessage ? c.lastMessage.text : $t('dashboard.discussion.noMessages') }}
					</p>
				</div>
			</li>
		</ul>
	</section>
</template>

<script setup lang="ts">
import {ref, onMounted} from 'vue'
import {useRouter} from 'vue-router'
import {useI18n} from 'vue-i18n'

import ConversationService from '@/services/conversation'
import {getDisplayName} from '@/models/user'
import {formatDateSince} from '@/helpers/time/formatDate'
import {useAuthStore} from '@/stores/auth'
import type {IConversation} from '@/modelTypes/IConversation'

const router = useRouter()
const {t} = useI18n()
const authStore = useAuthStore()

const loading = ref(true)
const conversations = ref<IConversation[]>([])

function conversationTitle(c: IConversation): string {
	if (c.title) return c.title
	const other = c.members.find(m => m.id !== authStore.info?.id)
	return other ? getDisplayName(other) : t('discussion.memberChat')
}

function initials(name: string): string {
	return name.split(/\s+/).filter(Boolean).slice(0, 2).map(p => p[0]?.toUpperCase() ?? '').join('')
}

function goToConversation() {
	router.push({name: 'discussion.index'})
}

onMounted(async () => {
	try {
		const all = await new ConversationService().getAll()
		conversations.value = all
			.filter(c => c.lastMessage !== null)
			.sort((a, b) => new Date(b.lastMessage!.created).getTime() - new Date(a.lastMessage!.created).getTime())
			.slice(0, 6)
	} finally {
		loading.value = false
	}
})
</script>

<style lang="scss" scoped>
.recent-discussion {
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: 12px;
	box-shadow: 0 2px 4px rgba(0, 0, 0, .03);
	padding: 1.25rem;

	&__head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-block-end: 1rem;

		h2 {
			font-size: 1rem;
			font-weight: 600;
			color: var(--grey-900);
			margin: 0;
		}
	}

	&__link {
		display: inline-flex;
		align-items: center;
		gap: .3rem;
		font-size: .8rem;
		font-weight: 600;
		color: var(--primary);
	}

	&__empty {
		font-size: .85rem;
		color: var(--grey-400);
		font-style: italic;
		padding-block: .5rem;
	}

	&__list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: .9rem;
	}

	&__item {
		display: flex;
		gap: .75rem;
		cursor: pointer;
	}

	&__avatar {
		display: grid;
		place-items: center;
		inline-size: 2.25rem;
		block-size: 2.25rem;
		border-radius: 50%;
		background: var(--primary);
		color: #ffffff;
		font-size: .7rem;
		font-weight: 700;
		flex-shrink: 0;
	}

	&__meta {
		flex: 1;
		min-inline-size: 0;
	}

	&__top {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: .5rem;
	}

	&__name {
		font-size: .85rem;
		font-weight: 600;
		color: var(--grey-900);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	&__date {
		font-size: .68rem;
		color: var(--grey-400);
		flex-shrink: 0;
	}

	&__snippet {
		font-size: .8rem;
		color: var(--grey-500);
		margin: .15rem 0 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
}
</style>
