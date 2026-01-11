(() => {
  const basename = (p) => {
    if (typeof p !== "string") {
      return "";
    }
    const idx = p.lastIndexOf("/");
    return idx >= 0 ? p.slice(idx + 1) : p;
  };

  const setBackground = (url) => {
    if (!url) {
      return;
    }
    document.documentElement.style.setProperty("--mtg-bg-image", `url("${url}")`);
  };

  const pickRandom = (arr) => {
    if (!Array.isArray(arr) || arr.length === 0) {
      return "";
    }
    return arr[Math.floor(Math.random() * arr.length)];
  };

  const load = async () => {
    try {
      const res = await fetch("/img/index.json", { cache: "no-store" });
      if (!res.ok) {
        return;
      }
      const body = await res.json();
      const images = body && Array.isArray(body.images) ? body.images : [];
      const players = images.filter((p) => /^player\d+\./i.test(basename(p)));
      setBackground(pickRandom(players.length ? players : images));
    } catch (err) {}
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", load);
  } else {
    load();
  }
})();
