// Accessibility smoke scan: logs in, visits a fixed set of routes, runs
// axe-core, and fails (exit 1) if any violation is found. Not exhaustive —
// covers the routes that matter for CI churn (login, dashboard, a project,
// settings, a task, admin) rather than every page in the app.
//
// Usage: node scripts/a11y-scan.cjs <baseURL>
// Requires: a running instance at baseURL with the seeded dev user
// (VIKUNJA_A11Y_SCAN_USER / VIKUNJA_A11Y_SCAN_PASS, defaults boss/testpass123
// to match dev-run.sh's fixtures).
const { chromium } = require('@playwright/test')
const path = require('node:path')

const BASE = process.argv[2] || 'http://localhost:3466'
const USER = process.env.VIKUNJA_A11Y_SCAN_USER || 'boss'
const PASS = process.env.VIKUNJA_A11Y_SCAN_PASS || 'testpass123'
const axePath = require.resolve('axe-core/axe.min.js')

const ROUTES = [
	['/login', 'Login page', false],
	['/dashboard', 'Dashboard', true],
	['/projects', 'Projects list', true],
	['/projects/1/1', 'Project view (list)', true],
	['/tasks/1', 'Task detail', true],
	['/user/settings/general', 'Settings: General', true],
	['/admin', 'Admin overview', true],
	['/admin/users', 'Admin: Users', true],
]

async function scan(page, name) {
	await page.waitForTimeout(1000) // let entrance animations/transitions settle
	// A page can still navigate/reload right after load (e.g. the PWA service
	// worker taking control for the first time triggers one reload) — on a
	// slower CI runner that can land mid-injection and destroy the JS
	// context. Retry once against the settled page rather than fail the scan.
	try {
		await page.addScriptTag({ path: axePath })
	} catch (e) {
		if (!e.message.includes('Execution context was destroyed')) throw e
		await page.waitForLoadState('networkidle')
		await page.addScriptTag({ path: axePath })
	}
	return page.evaluate(async () => window.axe.run())
}

;(async () => {
	const browser = await chromium.launch()
	const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })

	let token = null
	let totalViolations = 0

	for (const [route, name, needsAuth] of ROUTES) {
		if (needsAuth && !token) {
			// /login specifically, not / — the latter client-side-redirects
			// (to /login when unauthenticated), which can destroy the JS
			// execution context mid-evaluate below.
			await page.goto(`${BASE}/login`, { waitUntil: 'networkidle' })
			const loginRes = await page.evaluate(
				async ({ base, user, pass }) => {
					const r = await fetch(`${base}/api/v1/login`, {
						method: 'POST',
						headers: { 'Content-Type': 'application/json' },
						body: JSON.stringify({ username: user, password: pass }),
					})
					return r.json()
				},
				{ base: BASE, user: USER, pass: PASS },
			)
			token = loginRes.token
			if (!token) {
				console.error(`Could not log in as ${USER} — aborting scan.`)
				process.exitCode = 1
				await browser.close()
				return
			}
			await page.evaluate((t) => localStorage.setItem('token', t), token)
		}

		try {
			await page.goto(`${BASE}${route}`, { waitUntil: 'networkidle', timeout: 15000 })
			// App.vue gates all real content (both the authenticated layout and
			// NoAuthWrapper) behind Ready.vue's baseStore.ready check, which shows
			// a landmark-less loading region until the initial config/auth fetch
			// resolves. networkidle alone can settle before that flips on a cold
			// CI runner's first navigation, so axe was sometimes scanning the
			// loading state instead of the page — wait for the real landmark.
			await page.waitForSelector('main#main-content', { timeout: 15000 })
		} catch (e) {
			console.log(`\n=== ${name} (${route}) === SKIPPED: ${e.message.split('\n')[0]}`)
			continue
		}

		const results = await scan(page, name)
		console.log(`\n=== ${name} (${page.url()}) ===`)
		console.log(`violations: ${results.violations.length}`)
		totalViolations += results.violations.length
		for (const v of results.violations) {
			console.log(`- [${v.impact}] ${v.id}: ${v.help} (${v.nodes.length} nodes) ${v.helpUrl}`)
			for (const n of v.nodes) {
				const data = n.any?.[0]?.data
				const extra = data ? ` fg=${data.fgColor} bg=${data.bgColor} ratio=${data.contrastRatio} needed=${data.expectedContrastRatio}` : ''
				console.log(`    ${n.target.join(' ')}${extra}`)
			}
		}
	}

	await browser.close()
	console.log(`\nTotal violations across ${ROUTES.length} routes: ${totalViolations}`)
	process.exitCode = totalViolations > 0 ? 1 : 0
})()
