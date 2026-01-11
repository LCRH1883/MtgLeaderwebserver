package httpapi

import (
	"html/template"
	"net/http"
)

const (
	androidTestingURL = "https://play.google.com/apps/testing/com.intagri.mtgleader"
	androidStoreURL   = "https://play.google.com/store/apps/details?id=com.intagri.mtgleader"
	privacyUpdated    = "2025-12-29"
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

func (a *api) handleWikiDeleteAccount(w http.ResponseWriter, r *http.Request) {
	renderPublicPage(w, http.StatusOK, "Wiki: Delete Account", publicWikiDeleteAccountBody)
}

func renderPublicPage(w http.ResponseWriter, status int, title string, body template.HTML) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
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
    <span class="inline-flex items-center rounded-full border border-rose-500/30 bg-rose-500/10 px-3 py-1 text-xs font-semibold uppercase tracking-[0.25em] text-rose-200">Beta</span>
    <h1 class="font-['Space_Grotesk'] text-4xl font-bold tracking-tight text-slate-50 sm:text-5xl">MTG Leader is building the next-gen hub for Magic: The Gathering playgroups.</h1>
    <p class="text-sm leading-6 text-slate-300">Track matches, manage friends, and surface stats that matter for your playgroup. The Android app, iOS app, and backend server are all in beta, with a full-feature release planned for late February.</p>
    <div class="flex flex-wrap gap-3 pt-1">
      <a class="inline-flex items-center justify-center rounded-xl bg-rose-600 px-5 py-3 text-sm font-semibold text-white shadow-sm hover:bg-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-400/40" href="/app/login">User Login</a>
      <a class="inline-flex items-center justify-center rounded-xl border border-white/10 bg-white/5 px-5 py-3 text-sm font-semibold text-slate-50 shadow-sm hover:border-white/20 hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-slate-300/30" href="` + androidTestingURL + `" target="_blank" rel="noopener">Android Test</a>
      <a class="inline-flex items-center justify-center rounded-xl border border-white/10 bg-white/5 px-5 py-3 text-sm font-semibold text-slate-50 shadow-sm hover:border-white/20 hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-slate-300/30" href="` + androidStoreURL + `" target="_blank" rel="noopener">Play Store</a>
    </div>
    <div class="rounded-2xl border border-rose-500/20 bg-rose-500/10 px-4 py-4 text-sm font-semibold text-rose-100">Android + iOS apps and the backend server are in beta. Full release is planned for late February.</div>
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
    <li>Sign in at <a class="font-semibold text-rose-200 hover:text-rose-100" href="/app/login">/app/login</a>.</li>
    <li>Open your profile at <a class="font-semibold text-rose-200 hover:text-rose-100" href="/app/profile">/app/profile</a>.</li>
    <li>Scroll to the "Delete account" panel.</li>
    <li>Type <strong>DELETE</strong> in the confirmation field.</li>
    <li>Click "Delete account" to confirm.</li>
  </ul>

  <p class="mt-6 text-sm leading-6 text-slate-300">If you do not see the delete panel, refresh the page or contact support.</p>
  <p class="mt-2 text-sm leading-6 text-slate-300">Support: <a class="font-semibold text-rose-200 hover:text-rose-100" href="mailto:contact@mtgleader.xyz">contact@mtgleader.xyz</a></p>
</section>
`)
