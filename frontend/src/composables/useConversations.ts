import {ref, computed, onMounted, onUnmounted} from 'vue'

import ConversationService from '@/services/conversation'
import type {IConversation} from '@/modelTypes/IConversation'

const POLL_INTERVAL = 10000

// Polls the caller's conversation list. Shared by the sidebar nav (an
// unread dot next to "Discussion", visible even when that page isn't open)
// and the Discussion view itself — each gets its own independent poll, same
// as useOrgUsers, rather than a shared store; conversation lists are small
// and this matches the existing convention here.
export function useConversations() {
	const loading = ref(true)
	const conversations = ref<IConversation[]>([])

	async function load() {
		conversations.value = await new ConversationService().getAll()
	}

	let poll: ReturnType<typeof setInterval>

	onMounted(async () => {
		try {
			await load()
		} finally {
			loading.value = false
		}
		poll = setInterval(load, POLL_INTERVAL)
	})

	onUnmounted(() => clearInterval(poll))

	const hasUnread = computed(() => conversations.value.some(c => c.isUnread))

	return {loading, conversations, hasUnread, load}
}
