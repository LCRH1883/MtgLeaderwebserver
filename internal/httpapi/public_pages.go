package httpapi

import (
	"html/template"
	"net/http"
)

const (
	androidStoreURL = "https://play.google.com/store/apps/details?id=com.intagri.mtgleader"
	appleStoreURL   = "https://apps.apple.com/app/mtg-leader/id6757661562"
	privacyUpdated  = "2025-12-29"
)

var publicPageT = template.Must(template.New("public").Parse(publicLayout))

type publicPageData struct {
	Title string
	Body  template.HTML
}

func (a *api) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	renderPublicPage(w, http.StatusOK, "MTG Leader", publicHomeBody)
}

func (a *api) handlePrivacyWeb(w http.ResponseWriter, r *http.Request) {
	renderPublicPage(w, http.StatusOK, "Web Privacy Policy", publicPrivacyWebBody)
}

func (a *api) handlePrivacyAndroid(w http.ResponseWriter, r *http.Request) {
	renderPublicPage(w, http.StatusOK, "Android Privacy Policy", publicPrivacyAndroidBody)
}

func (a *api) handlePrivacyApple(w http.ResponseWriter, r *http.Request) {
	renderPublicPage(w, http.StatusOK, "Apple Privacy Policy", publicPrivacyAppleBody)
}

func (a *api) handleWikiIndex(w http.ResponseWriter, r *http.Request) {
	renderPublicPage(w, http.StatusOK, "Wiki", publicWikiIndexBody)
}

func (a *api) handleWikiDeleteAccount(w http.ResponseWriter, r *http.Request) {
	renderPublicPage(w, http.StatusOK, "Wiki: Delete Account", publicWikiDeleteAccountBody)
}

func (a *api) handleWikiWebUI(w http.ResponseWriter, r *http.Request) {
	renderPublicPage(w, http.StatusOK, "Wiki: Web UI Guide", publicWikiWebUIBody)
}

func (a *api) handleWikiMobileApp(w http.ResponseWriter, r *http.Request) {
	renderPublicPage(w, http.StatusOK, "Wiki: Mobile App Guide", publicWikiMobileAppBody)
}

func renderPublicPage(w http.ResponseWriter, status int, title string, body template.HTML) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = publicPageT.Execute(w, publicPageData{
		Title: title,
		Body:  body,
	})
}

const publicLayout = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width,initial-scale=1" />
    <title>{{.Title}}</title>
    <script type="module">
      import "https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4";
    </script>
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@500;700&family=Work+Sans:wght@400;500;600&display=swap" rel="stylesheet" />
    <link rel="icon" type="image/png" sizes="32x32" href="/icon/favicon-32.png" />
    <link rel="icon" type="image/png" sizes="16x16" href="/icon/favicon-16.png" />
    <link rel="icon" type="image/png" sizes="192x192" href="/icon/android-chrome-192.png" />
    <link rel="apple-touch-icon" sizes="180x180" href="/icon/apple-touch-icon.png" />
    <meta name="theme-color" content="#5a0a0f" />
    <link rel="stylesheet" href="/app/static/bg.css?v=1" />
    <link rel="stylesheet" href="/app/static/gothic.css?v=1" />
    <script defer src="/app/static/bg.js?v=1"></script>
  </head>
  <body class="min-h-screen bg-gradient-to-br from-slate-950 via-slate-950 to-slate-900 text-slate-50 antialiased" style="font-family:'Work Sans','Helvetica Neue',Arial,sans-serif;">
    <header class="sticky top-0 z-10 border-b border-white/10 bg-slate-950/60 backdrop-blur">
      <div class="mx-auto flex w-full max-w-6xl flex-col gap-4 px-6 py-5 sm:flex-row sm:items-center sm:justify-between">
        <a class="flex items-center gap-3 no-underline" href="/">
          <img class="h-11 w-11 rounded-2xl object-cover shadow-sm ring-1 ring-white/10" src="/img/wizard_icon.png" width="44" height="44" alt="MTG Leader" />
          <span class="leading-tight">
            <span class="block font-['Space_Grotesk'] text-lg font-bold text-slate-50">MTG Leader</span>
            <span class="block text-sm text-slate-300">Magic: The Gathering playgroup</span>
          </span>
        </a>
        <nav class="flex flex-wrap items-center gap-3">
          <a class="inline-flex items-center rounded-full bg-rose-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-400/40" href="/app/login">User Login</a>
          <a class="inline-flex items-center rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-slate-50 shadow-sm hover:border-white/20 hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-slate-300/30" href="/wiki">Wiki</a>
        </nav>
      </div>
    </header>
    <main class="mx-auto w-full max-w-6xl px-6 pb-16 pt-10">
      {{.Body}}
      <footer class="mt-14 border-t border-white/10 pt-6 text-sm text-slate-300">
        <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>Copyright 2025 MTG Leader. Built for Magic: The Gathering Table Top Games.</div>
          <div class="flex flex-wrap gap-x-4 gap-y-2">
            <a class="font-semibold text-rose-200 hover:text-rose-100" href="/privacy">Web privacy</a>
            <a class="font-semibold text-rose-200 hover:text-rose-100" href="/privacy/android">Android privacy</a>
            <a class="font-semibold text-rose-200 hover:text-rose-100" href="/privacy/apple">Apple privacy</a>
            <a class="font-semibold text-rose-200 hover:text-rose-100" href="/wiki">Account deletion</a>
            <a class="font-semibold text-rose-200 hover:text-rose-100" href="/app/login">User login</a>
          </div>
        </div>
      </footer>
    </main>
  </body>
</html>`

var publicHomeBody = template.HTML(`
<section class="grid grid-cols-1 gap-8 lg:grid-cols-2 lg:items-start">
  <div class="space-y-5">
    <span class="inline-flex items-center rounded-full border border-rose-500/30 bg-rose-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.25em] text-rose-200">Now live on iOS + Android</span>
    <h1 class="font-['Space_Grotesk'] text-4xl font-bold tracking-tight text-slate-50 sm:text-5xl">MTG Leader is building the next-gen hub for Magic: The Gathering playgroups.</h1>
    <p class="text-sm leading-6 text-slate-300">Track matches, manage friends, and surface stats that matter for your playgroup. Download the mobile app on Google Play or the App Store.</p>
    <div class="flex flex-wrap gap-3 pt-1">
      <a class="inline-flex items-center justify-center rounded-xl bg-rose-600 px-5 py-3 text-sm font-semibold text-white shadow-sm hover:bg-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-400/40" href="/app/login">User Login</a>
      <a class="inline-flex items-center justify-center gap-2 rounded-xl border border-white/10 bg-white/5 px-5 py-3 text-sm font-semibold text-slate-50 shadow-sm hover:border-white/20 hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-slate-300/30" href="` + androidStoreURL + `" target="_blank" rel="noopener">
        <svg class="h-5 w-5 text-emerald-300" viewBox="0 0 24 24" aria-hidden="true">
          <path fill="currentColor" d="M17.6 9.48l1.84-3.18c.16-.31.04-.7-.26-.86a.65.65 0 0 0-.86.25l-1.88 3.25c-1.02-.47-2.15-.74-3.36-.74s-2.34.27-3.36.74L7.9 5.69a.65.65 0 0 0-.86-.25.65.65 0 0 0-.25.86l1.84 3.18A5.99 5.99 0 0 0 6 14.5V18c0 .55.45 1 1 1h1v3c0 .55.45 1 1 1s1-.45 1-1v-3h6v3c0 .55.45 1 1 1s1-.45 1-1v-3h1c.55 0 1-.45 1-1v-3.5c0-2.15-1.11-4.05-2.4-5.02zM9 13.5c-.55 0-1-.45-1-1s.45-1 1-1 1 .45 1 1-.45 1-1 1zm8 0c-.55 0-1-.45-1-1s.45-1 1-1 1 .45 1 1-.45 1-1 1z"/>
        </svg>
        Android Download
      </a>
      <a class="inline-flex items-center justify-center gap-2 rounded-xl border border-white/10 bg-white/5 px-5 py-3 text-sm font-semibold text-slate-50 shadow-sm hover:border-white/20 hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-slate-300/30" href="` + appleStoreURL + `" target="_blank" rel="noopener">
        <svg class="h-5 w-5 text-slate-200" viewBox="0 0 24 24" aria-hidden="true">
          <path fill="currentColor" d="M16.365 1.43c0 1.14-.42 2.22-1.26 3.24-.96 1.17-2.4 2.07-3.87 1.95-.12-1.14.48-2.34 1.32-3.3.96-1.08 2.52-1.86 3.81-1.89zm5.52 16.86c-.3.69-.66 1.35-1.14 1.98-.66.9-1.2 1.53-1.62 1.89-.66.6-1.35.9-2.1.9-.54 0-1.2-.15-1.98-.45-.78-.3-1.5-.45-2.19-.45-.72 0-1.47.15-2.25.45-.78.3-1.41.45-1.89.48-.72.03-1.44-.3-2.16-.96-.45-.39-1.02-1.05-1.71-1.98-.75-1.02-1.38-2.22-1.86-3.6-.51-1.5-.78-2.94-.78-4.35 0-1.62.33-3.03 1.02-4.2.54-.93 1.26-1.68 2.19-2.25.93-.57 1.92-.87 2.97-.9.57 0 1.32.18 2.25.54.93.36 1.53.54 1.8.54.21 0 .9-.21 2.07-.63 1.11-.39 2.04-.54 2.82-.48 2.07.18 3.63.99 4.65 2.46-1.86 1.14-2.79 2.73-2.76 4.77.03 1.59.57 2.91 1.62 3.96.48.51 1.02.9 1.62 1.2z"/>
        </svg>
        iOS Download
      </a>
    </div>
    <div class="rounded-2xl border border-rose-500/20 bg-rose-500/10 px-4 py-4 text-sm font-semibold text-rose-100">Now available on iOS and Android.</div>
  </div>
  <div class="rounded-3xl border border-white/10 bg-white/5 p-6 shadow-sm">
    <h2 class="font-['Space_Grotesk'] text-2xl font-bold tracking-tight text-slate-50">What is MTG Leader?</h2>
    <p class="mt-3 text-sm leading-6 text-slate-300">A focused app for Magic: The Gathering players who want to keep track of their playgroup, track match outcomes, and see who is on top. The apps and backend are being developed together during beta.</p>
    <div class="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
      <div class="rounded-2xl border border-white/10 bg-slate-950/40 p-4">
        <h3 class="font-['Space_Grotesk'] text-lg font-bold text-slate-50">Playgroup management</h3>
        <p class="mt-2 text-sm leading-6 text-slate-300">Add friends, track who is in your regular group, and keep everyone connected.</p>
      </div>
      <div class="rounded-2xl border border-white/10 bg-slate-950/40 p-4">
        <h3 class="font-['Space_Grotesk'] text-lg font-bold text-slate-50">Match tracking</h3>
        <p class="mt-2 text-sm leading-6 text-slate-300">Capture results, placements, and formats to build a shared record.</p>
      </div>
      <div class="rounded-2xl border border-white/10 bg-slate-950/40 p-4">
        <h3 class="font-['Space_Grotesk'] text-lg font-bold text-slate-50">Stats overview</h3>
        <p class="mt-2 text-sm leading-6 text-slate-300">See win rates and trends across Commander, Brawl, Standard, and Modern.</p>
      </div>
    </div>
  </div>
</section>
`)

var publicPrivacyWebBody = template.HTML(`
<section class="rounded-3xl border border-white/10 bg-white/5 p-6 shadow-sm">
  <span class="inline-flex items-center rounded-full border border-rose-500/30 bg-rose-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.25em] text-rose-200">Web Privacy Policy</span>
  <h1 class="mt-4 font-['Space_Grotesk'] text-3xl font-bold tracking-tight text-slate-50">MTG Leader Web Privacy Policy</h1>
  <p class="mt-2 text-sm text-slate-300">Last updated: ` + privacyUpdated + `</p>
  <p class="mt-4 text-sm leading-6 text-slate-300">MTG Leader is in beta for Magic: The Gathering players. This policy describes how the web experience handles data.</p>

  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Data we collect</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Account information such as email, username, and password (stored as a secure hash).</li>
    <li>Profile details you provide, like display name and avatar.</li>
    <li>Gameplay data such as matches, placements, friends, and stats.</li>
    <li>Session and security data, including cookies needed to keep you signed in.</li>
  </ul>

  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">How we use data</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>To authenticate your account and keep you signed in.</li>
    <li>To power app features like matches, friends, and stats.</li>
    <li>To protect the service and prevent abuse.</li>
  </ul>

  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Sharing</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">We do not sell your personal data. We share data only with infrastructure providers needed to run the service.</p>
  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Retention</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">We keep data for as long as your account is active or as required for service operation. You can request deletion.</p>
  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Your choices</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">You can update profile information in the app. For deletion requests, use the support contact listed in the app or store listing.</p>
</section>
`)

var publicPrivacyAndroidBody = template.HTML(`
<section class="rounded-3xl border border-white/10 bg-white/5 p-6 shadow-sm">
  <span class="inline-flex items-center rounded-full border border-rose-500/30 bg-rose-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.25em] text-rose-200">Android Privacy Policy</span>
  <h1 class="mt-4 font-['Space_Grotesk'] text-3xl font-bold tracking-tight text-slate-50">MTG Leader Android Privacy Policy</h1>
  <p class="mt-2 text-sm text-slate-300">Last updated: ` + privacyUpdated + `</p>
  <p class="mt-4 text-sm leading-6 text-slate-300">The Android app connects to the same MTG Leader backend as the web experience. Android, iOS, and the backend server are currently in beta.</p>

  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Data we collect</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Account data such as email, username, and password (stored as a secure hash).</li>
    <li>Profile data like display name and avatar.</li>
    <li>Gameplay data including matches, placements, friends, and stats.</li>
    <li>Operational diagnostics provided by Google Play reporting tools.</li>
  </ul>

  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">How we use data</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>To run core features like login, friends, and match tracking.</li>
    <li>To improve app stability and performance.</li>
    <li>To secure the service and prevent abuse.</li>
  </ul>

  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Sharing</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">We do not sell personal data. Data is shared only with infrastructure providers and Google Play services required to deliver the app.</p>
  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Retention</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">Data is retained while your account is active or as required for service operations. You can request deletion.</p>
  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Your choices</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">You can update profile information in the app. For deletion requests, use the support contact listed in the Play Store entry.</p>
</section>
`)

var publicPrivacyAppleBody = template.HTML(`
<section class="rounded-3xl border border-white/10 bg-white/5 p-6 shadow-sm">
  <span class="inline-flex items-center rounded-full border border-rose-500/30 bg-rose-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.25em] text-rose-200">Apple Privacy Policy</span>
  <h1 class="mt-4 font-['Space_Grotesk'] text-3xl font-bold tracking-tight text-slate-50">MTG Leader Apple Privacy Policy</h1>
  <p class="mt-2 text-sm text-slate-300">Last updated: ` + privacyUpdated + `</p>
  <p class="mt-4 text-sm leading-6 text-slate-300">The iOS app connects to the same MTG Leader backend as the web experience. The iOS app and backend server are currently in beta.</p>

  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Data we collect</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Account data such as email, username, and password (stored as a secure hash).</li>
    <li>If you use Sign in with Apple, we receive an Apple identifier and (depending on your settings) an email address.</li>
    <li>Profile data like display name and avatar.</li>
    <li>Gameplay data including matches, placements, friends, and stats.</li>
    <li>Operational diagnostics provided by Apple/Device crash reporting and App Store reporting tools.</li>
  </ul>

  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">How we use data</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>To run core features like login, friends, and match tracking.</li>
    <li>To improve app stability and performance.</li>
    <li>To secure the service and prevent abuse.</li>
  </ul>

  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Sharing</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">We do not sell personal data. Data is shared only with infrastructure providers and Apple services required to deliver the app (such as Sign in with Apple, notifications, and App Store reporting).</p>
  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Retention</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">Data is retained while your account is active or as required for service operations. You can request deletion.</p>
  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Your choices</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">You can update profile information in the app. For deletion requests, use the support contact listed in the App Store entry.</p>

  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Contact</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">Questions or deletion requests: <a class="font-semibold text-rose-200 hover:text-rose-100" href="mailto:contact@mtgleader.xyz">contact@mtgleader.xyz</a>.</p>
</section>
`)

var publicWikiDeleteAccountBody = template.HTML(`
<section class="rounded-3xl border border-white/10 bg-white/5 p-6 shadow-sm">
  <span class="inline-flex items-center rounded-full border border-rose-500/30 bg-rose-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.25em] text-rose-200">Wiki</span>
  <h1 class="mt-4 font-['Space_Grotesk'] text-3xl font-bold tracking-tight text-slate-50">Delete your account</h1>
  <p class="mt-4 text-sm leading-6 text-slate-300">You can permanently delete your MTG Leader account from the web profile page. This removes your profile, matches, and friends.</p>

  <h2 class="mt-8 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Steps</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Sign in at <a class="font-semibold text-rose-200 hover:text-rose-100" href="/app/login">Login</a>.</li>
    <li>Open your profile at <a class="font-semibold text-rose-200 hover:text-rose-100" href="/app/profile">Profile</a>.</li>
    <li>Scroll to the "Delete account" panel.</li>
    <li>Type <strong>DELETE</strong> in the confirmation field.</li>
    <li>Click "Delete account" to confirm.</li>
  </ul>

  <p class="mt-6 text-sm leading-6 text-slate-300">If you do not see the delete panel, refresh the page or contact support.</p>
	  <p class="mt-2 text-sm leading-6 text-slate-300">Support: <a class="font-semibold text-rose-200 hover:text-rose-100" href="mailto:contact@mtgleader.xyz">contact@mtgleader.xyz</a></p>
	</section>
	`)

var publicWikiIndexBody = template.HTML(`
<section class="rounded-3xl border border-white/10 bg-white/5 p-6 shadow-sm">
  <span class="inline-flex items-center rounded-full border border-rose-500/30 bg-rose-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.25em] text-rose-200">Wiki</span>
  <h1 class="mt-4 font-['Space_Grotesk'] text-3xl font-bold tracking-tight text-slate-50">MTG Leader wiki</h1>
  <p class="mt-4 text-sm leading-6 text-slate-300">Short guides for using MTG Leader on web and mobile.</p>

  <div class="mt-8 grid grid-cols-1 gap-4 sm:grid-cols-2">
    <a class="group rounded-3xl border border-white/10 bg-white/5 p-6 shadow-sm transition hover:-translate-y-0.5 hover:border-white/20 hover:bg-white/10" href="/wiki/mobile-app">
      <div class="font-['Space_Grotesk'] text-xl font-bold text-slate-50">Mobile app guide</div>
      <div class="mt-2 text-sm leading-6 text-slate-300">How to use the iOS/Android app (setup, gameplay, counters, friends, stats, notifications).</div>
      <div class="mt-4 text-sm font-semibold text-rose-200 group-hover:text-rose-100">Open guide →</div>
    </a>
    <a class="group rounded-3xl border border-white/10 bg-white/5 p-6 shadow-sm transition hover:-translate-y-0.5 hover:border-white/20 hover:bg-white/10" href="/wiki/web-ui">
      <div class="font-['Space_Grotesk'] text-xl font-bold text-slate-50">Web UI guide</div>
      <div class="mt-2 text-sm leading-6 text-slate-300">How to use the user web app at <code class="rounded bg-white/10 px-1.5 py-0.5 text-slate-100">/app</code> (friends, matches, stats, profile).</div>
      <div class="mt-4 text-sm font-semibold text-rose-200 group-hover:text-rose-100">Open guide →</div>
    </a>
    <a class="group rounded-3xl border border-white/10 bg-white/5 p-6 shadow-sm transition hover:-translate-y-0.5 hover:border-white/20 hover:bg-white/10" href="/wiki/delete-account">
      <div class="font-['Space_Grotesk'] text-xl font-bold text-slate-50">Delete your account</div>
      <div class="mt-2 text-sm leading-6 text-slate-300">How to permanently delete your account from the profile page.</div>
      <div class="mt-4 text-sm font-semibold text-rose-200 group-hover:text-rose-100">Open guide →</div>
    </a>
  </div>

  <div class="mt-8 rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-slate-300">
    <div class="font-semibold text-slate-100">Quick links</div>
    <ul class="mt-2 list-disc space-y-1 pl-6">
      <li><a class="font-semibold text-rose-200 hover:text-rose-100" href="/app/login">Sign in</a></li>
      <li><a class="font-semibold text-rose-200 hover:text-rose-100" href="/app/register">Create account</a></li>
    </ul>
  </div>
</section>
`)

var publicWikiWebUIBody = template.HTML(`
<section class="rounded-3xl border border-white/10 bg-white/5 p-6 shadow-sm">
  <span class="inline-flex items-center rounded-full border border-rose-500/30 bg-rose-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.25em] text-rose-200">Wiki</span>
  <h1 class="mt-4 font-['Space_Grotesk'] text-3xl font-bold tracking-tight text-slate-50">Web UI guide</h1>
  <p class="mt-4 text-sm leading-6 text-slate-300">MTG Leader includes a user web app that runs under <code class="rounded bg-white/10 px-1.5 py-0.5 text-slate-100">/app</code>. It focuses on managing friends, viewing matches, and checking stats.</p>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Getting started</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Open <a class="font-semibold text-rose-200 hover:text-rose-100" href="/app/login">Login</a> to sign in.</li>
    <li>New users can create an account at <a class="font-semibold text-rose-200 hover:text-rose-100" href="/app/register">Register</a>.</li>
    <li>After signing in, you land on <a class="font-semibold text-rose-200 hover:text-rose-100" href="/app/">Home</a> and can use the top navigation bar to move between pages.</li>
  </ul>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Navigation & basics</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li><strong>Home</strong> (<code class="rounded bg-white/10 px-1.5 py-0.5 text-slate-100">/app/</code>) is a quick launcher to Friends, Matches, Stats, and Profile.</li>
    <li><strong>Dark mode</strong> is a toggle in the top bar; your choice is saved in your browser.</li>
    <li><strong>Profile</strong> opens your account settings and avatar tools.</li>
    <li><strong>Logout</strong> ends your session on this device.</li>
  </ul>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Friends</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">Open <a class="font-semibold text-rose-200 hover:text-rose-100" href="/app/friends">Friends</a> to search players and manage friend requests.</p>
  <div class="mt-4 space-y-3 text-sm leading-6 text-slate-300">
    <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
      <div class="font-semibold text-slate-100">Search players</div>
      <ul class="mt-2 list-disc space-y-1 pl-6">
        <li>Use the search field to look up usernames (minimum 3 characters).</li>
        <li>From the results list you can send a request (“Add friend”).</li>
        <li>If you already have a pending request, you will see “Pending”/“Cancel request” instead.</li>
      </ul>
    </div>
    <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
      <div class="font-semibold text-slate-100">Incoming & outgoing requests</div>
      <ul class="mt-2 list-disc space-y-1 pl-6">
        <li><strong>Incoming</strong>: accept or decline requests from other users.</li>
        <li><strong>Outgoing</strong>: view requests you sent and cancel them.</li>
      </ul>
    </div>
    <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
      <div class="font-semibold text-slate-100">Friends list & head-to-head stats</div>
      <ul class="mt-2 list-disc space-y-1 pl-6">
        <li>Your accepted friends appear in the friends list.</li>
        <li>If you have match history, the page may show “Friend stats” (wins/losses/co-losses) against each friend.</li>
      </ul>
    </div>
  </div>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Matches</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">Open <a class="font-semibold text-rose-200 hover:text-rose-100" href="/app/matches">Matches</a> to review your most recent matches.</p>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>The list shows date played, format, number of players, total duration, turn count, and the winner.</li>
    <li>Click a match to open details at <code class="rounded bg-white/10 px-1.5 py-0.5 text-slate-100">/app/matches/&lt;id&gt;</code> (placements, ranks, elimination turn/batch when available).</li>
  </ul>
  <p class="mt-3 text-sm leading-6 text-slate-300"><strong>Note:</strong> The web UI currently focuses on reviewing matches; creating/recording matches may happen through the mobile app or other clients connected to the same account.</p>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Stats</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">Open <a class="font-semibold text-rose-200 hover:text-rose-100" href="/app/stats">Stats</a> for a performance overview.</p>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li><strong>Overall</strong>: total matches, wins, losses, and average turn length.</li>
    <li><strong>Top opponents</strong>: who you beat most often, and who beats you most often.</li>
    <li><strong>By format</strong>: per-format breakdown (matches, wins, losses, average turn pace).</li>
  </ul>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Profile</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">Open <a class="font-semibold text-rose-200 hover:text-rose-100" href="/app/profile">Profile</a> to edit your public profile details.</p>
  <div class="mt-4 space-y-3 text-sm leading-6 text-slate-300">
    <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
      <div class="font-semibold text-slate-100">Display name</div>
      <ul class="mt-2 list-disc space-y-1 pl-6">
        <li>Set a display name (shown to friends). Leave it blank to hide it.</li>
        <li>Your username (for searches) is separate and cannot be changed here.</li>
      </ul>
    </div>
    <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
      <div class="font-semibold text-slate-100">Avatar cropper</div>
      <ul class="mt-2 list-disc space-y-1 pl-6">
        <li>Choose an image, then drag and zoom in the cropper.</li>
        <li>Click “Save avatar” to upload a 96×96 JPEG avatar for your account.</li>
      </ul>
    </div>
    <div class="rounded-2xl border border-white/10 bg-white/5 p-4">
      <div class="font-semibold text-slate-100">Delete account</div>
      <ul class="mt-2 list-disc space-y-1 pl-6">
        <li>Scroll to “Delete account”, type <strong>DELETE</strong>, and confirm to permanently remove your account.</li>
        <li>See the dedicated guide: <a class="font-semibold text-rose-200 hover:text-rose-100" href="/wiki/delete-account">Delete Account</a>.</li>
      </ul>
    </div>
  </div>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Password reset links</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">If you receive a password reset link, it will take you to <code class="rounded bg-white/10 px-1.5 py-0.5 text-slate-100">/app/reset?token=…</code>. Enter a new password (minimum 12 characters) and confirm.</p>

  <div class="mt-10 rounded-2xl border border-white/10 bg-white/5 p-4 text-sm text-slate-300">
    <div class="font-semibold text-slate-100">Troubleshooting</div>
    <ul class="mt-2 list-disc space-y-1 pl-6">
      <li>If you get redirected back to the login page, your session may have expired; sign in again.</li>
      <li>If a feature shows “unavailable”, the server may be in maintenance or missing required configuration; try again later or contact support.</li>
    </ul>
  </div>
</section>
`)

var publicWikiMobileAppBody = template.HTML(`
<section class="rounded-3xl border border-white/10 bg-white/5 p-6 shadow-sm">
  <span class="inline-flex items-center rounded-full border border-rose-500/30 bg-rose-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.25em] text-rose-200">Wiki</span>
  <h1 class="mt-4 font-['Space_Grotesk'] text-3xl font-bold tracking-tight text-slate-50">MTG Leader User Guide (Mobile)</h1>
  <p class="mt-4 text-sm leading-6 text-slate-300">MTG Leader is a Magic: The Gathering life and counter tracking app. You can play without logging in, and optional sign-in unlocks friends, notifications, and synced stats.</p>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Home and setup</h2>
  <ol class="mt-3 list-decimal space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Open the app. The home screen is the Setup screen.</li>
    <li>Set Players (1-8).</li>
    <li>Set Starting Life (20, 30, 40, or Custom).</li>
    <li>Choose a Tabletop layout. Options vary by player count; List layout is available for 2+ players.</li>
    <li>Optional: tap Customize to set player colors, decks, and assignments.</li>
    <li>Tap Start Game.</li>
  </ol>

  <div class="mt-4 rounded-2xl border border-white/10 bg-white/5 p-4 text-sm leading-6 text-slate-300">
    <div class="font-semibold text-slate-100">Top bar shortcuts</div>
    <ul class="mt-2 list-disc space-y-1 pl-6">
      <li>Friends icon (left)</li>
      <li>Profile (user icon)</li>
      <li>Settings (gear icon)</li>
      <li>Stats (crown icon)</li>
    </ul>
  </div>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Customize players (before the game)</h2>
  <ol class="mt-3 list-decimal space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Tap Customize from the Setup screen.</li>
    <li>Tap a player tile to open Customize Player.</li>
    <li>Deck: choose a profile (Default if you have not created others).</li>
    <li>Assign Player: pick a friend or select Unassigned.</li>
    <li>Temporary Name: enter a name for Unassigned players.</li>
    <li>Select Color: tap a color chip.</li>
    <li>Tap Save, then Start Game.</li>
  </ol>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Game screen basics</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Life and counters appear on each player panel.</li>
    <li>Tap + or - to change by 1. Press and hold to change faster.</li>
    <li>After selecting a starting player, tap the active player's life total to end their turn.</li>
    <li>List layout: players are stacked, and the wizard icon is in the top-left app bar.</li>
    <li>Tabletop layouts: a wizard button sits near the center and shows turn count and the timer.</li>
  </ul>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Turn flow</h2>
  <ol class="mt-3 list-decimal space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Open the game menu (wizard icon).</li>
    <li>Use Select Player and then tap a player panel, or choose Random Player.</li>
    <li>Turns advance by tapping the active player's life total.</li>
    <li>Use Turn to open the turn dialog:
      <ul class="mt-2 list-disc space-y-1 pl-6">
        <li>Go Back Turn</li>
        <li>Pause or Resume</li>
        <li>View the game clock</li>
      </ul>
    </li>
  </ol>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Dice</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">Open the game menu and select Dice. Choose Coin, D6, or D20. Results are shown immediately.</p>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Counters</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">Add and manage counters per player:</p>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Tabletop layouts: pull a player panel up or down to reveal actions, then tap Counters.</li>
    <li>List layout: tap the add-counter icon on the right side of the player row.</li>
    <li>Select counters from the grid; tap again to deselect.</li>
    <li>Tap Save to apply changes.</li>
    <li>Adjust counter values with + or -, and press and hold to change faster.</li>
    <li>If the rearrange icon is available, tap it to drag counters into a new order, then Save.</li>
  </ul>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Assign players during a match</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Tabletop layouts: pull a player panel up or down, then tap Assign Player.</li>
    <li>Choose a friend or Unassigned. If Unassigned, enter a Temporary Name.</li>
    <li>Avatars show when available.</li>
  </ul>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Game options (wizard menu → Options)</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Player Rotation: clockwise or counterclockwise.</li>
    <li>Turn Timer: toggle on/off and set minutes/seconds. When time hits 0, an alarm plays and the timer enters overtime.</li>
    <li>Keep Screen Awake: prevent the device from sleeping during the match.</li>
    <li>Hide Navigation: use immersive fullscreen.</li>
    <li>Record Match Data: enable or disable match recording.</li>
    <li>Auto Exit After Match: return to the previous screen after saving a match.</li>
  </ul>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Completing matches</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Use Complete Match in the game menu to save stats.</li>
    <li>When a player's life hits 0 or below, the app asks you to confirm elimination.</li>
    <li>When one player remains, the match completes automatically (and auto-exit if enabled).</li>
    <li>If Record Match Data is off, the app will not save or upload a match.</li>
  </ul>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Match history</h2>
  <ol class="mt-3 list-decimal space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Open Settings → Match History.</li>
    <li>Each row shows date, layout, duration, player count, and sync status (Pending/Synced/Failed).</li>
    <li>Tap a match for details and Retry Upload when needed.</li>
  </ol>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Stats</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Open from the home screen using the crown icon.</li>
    <li>View games played, wins, losses, and win percent.</li>
    <li>Head-to-head: tap a friend to view totals.</li>
    <li>Not logged in? You can still see local totals, but personal tracking is limited.</li>
  </ul>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Friends</h2>
  <ol class="mt-3 list-decimal space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Open Friends from the home screen or Settings.</li>
    <li>Add Friend: enter a username and tap Send Invite.</li>
    <li>Incoming Requests: Accept or Decline.</li>
    <li>Outgoing Requests: Cancel.</li>
    <li>Friends list: Remove.</li>
  </ol>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Notifications</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Open Settings → Notifications.</li>
    <li>Toggle friend request notifications.</li>
    <li>If notifications are blocked by the OS, the app will prompt you to enable them in system settings.</li>
  </ul>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Profile and account</h2>
  <ul class="mt-3 list-disc space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Open Profile from the home screen (user icon) or Settings → Profile.</li>
    <li>Log In, Sign Up, or use Forgot Password from the login screen.</li>
    <li>Update your display name and avatar (tap the avatar to select and crop).</li>
    <li>Upload matches on Wi‑Fi only: toggle in Profile for sync preference.</li>
    <li>Log Out clears local profile data and cached stats.</li>
  </ul>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Custom counters</h2>
  <ol class="mt-3 list-decimal space-y-2 pl-6 text-sm leading-6 text-slate-300">
    <li>Open Settings → Counters (login required).</li>
    <li>Tap Create Counter.</li>
    <li>Choose a counter type:
      <ul class="mt-2 list-disc space-y-1 pl-6">
        <li>Text/Emoji</li>
        <li>Local Image</li>
        <li>Image URL</li>
      </ul>
    </li>
    <li>Optional: enable Full Art for image-based counters.</li>
    <li>Set a Starting Value.</li>
    <li>Review the Preview and tap Save.</li>
    <li>Tap a counter to edit, or delete it if the delete icon is available.</li>
  </ol>
  <p class="mt-3 text-sm leading-6 text-slate-300">Custom counters are added to the Default deck profile for use in games.</p>

  <h2 class="mt-10 font-['Space_Grotesk'] text-xl font-bold text-slate-50">Decks</h2>
  <p class="mt-3 text-sm leading-6 text-slate-300">Settings → Manage Decks is currently a placeholder screen and is not available yet.</p>
</section>
`)
