/* DevRites landing — console choreography, reveals, nav, copy. Reduced-motion aware. */
(function () {
  "use strict";
  document.documentElement.classList.add("js");
  var reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  /* ---- nav: sticky shadow + mobile toggle ---- */
  var nav = document.getElementById("nav");
  if (nav) {
    var onScroll = function () { nav.classList.toggle("is-stuck", window.scrollY > 8); };
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });

    var burger = nav.querySelector(".nav__burger");
    if (burger) {
      var close = function () { nav.classList.remove("is-open"); burger.setAttribute("aria-expanded", "false"); };
      burger.addEventListener("click", function () {
        var open = nav.classList.toggle("is-open");
        burger.setAttribute("aria-expanded", open ? "true" : "false");
      });
      nav.querySelectorAll(".nav__links a").forEach(function (a) { a.addEventListener("click", close); });
      document.addEventListener("keydown", function (e) { if (e.key === "Escape") close(); });
    }
  }

  /* ---- scroll reveals (enhance already-visible content) ---- */
  var reveals = document.querySelectorAll(".reveal");
  if (reduced || !("IntersectionObserver" in window)) {
    reveals.forEach(function (el) { el.classList.add("in"); });
  } else {
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (en) {
        if (en.isIntersecting) { en.target.classList.add("in"); io.unobserve(en.target); }
      });
    }, { threshold: 0.12, rootMargin: "0px 0px -8% 0px" });
    reveals.forEach(function (el) { io.observe(el); });
  }

  /* ---- copy button ---- */
  var copyBtn = document.getElementById("copyBtn");
  if (copyBtn && navigator.clipboard) {
    var cmd = document.getElementById("installCmd");
    var label = copyBtn.querySelector("span");
    copyBtn.addEventListener("click", function () {
      navigator.clipboard.writeText(cmd.innerText.trim()).then(function () {
        copyBtn.classList.add("copied");
        var prev = label.textContent;
        label.textContent = "copied";
        setTimeout(function () { copyBtn.classList.remove("copied"); label.textContent = prev; }, 1800);
      });
    });
  }

  /* ---- hero console power-on ---- */
  var console_ = document.getElementById("console");
  var fill = document.getElementById("railFill");
  var nodes = console_ ? console_.querySelectorAll(".node") : [];
  var current = 2; // build = index 2 (3/5)
  var cmdEl = document.getElementById("termCmd");
  var cursor = document.getElementById("termCursor");
  var out = document.getElementById("termOut");
  var FULL = "/rite-build refresh-token";

  function finalState() {
    nodes.forEach(function (n, i) {
      n.classList.remove("is-done", "is-current");
      if (i < current) n.classList.add("is-done");
      else if (i === current) n.classList.add("is-current");
    });
    if (fill) fill.style.width = (current / (nodes.length - 1) * 100) + "%";
    if (cmdEl) cmdEl.textContent = FULL;
    if (cursor) cursor.style.display = "none";
    if (out) out.hidden = false;
  }

  function animate() {
    // 1) light nodes sequentially up to current
    var step = 0;
    var litTimer = setInterval(function () {
      if (step < current) { nodes[step].classList.add("is-done"); step++; }
      else {
        clearInterval(litTimer);
        nodes[current].classList.add("is-current");
        // 2) grow rail fill
        if (fill) fill.style.width = (current / (nodes.length - 1) * 100) + "%";
        // 3) type the command
        typeCmd();
      }
    }, 260);
  }

  function typeCmd() {
    var i = 0;
    var t = setInterval(function () {
      cmdEl.textContent = FULL.slice(0, ++i);
      if (i >= FULL.length) {
        clearInterval(t);
        setTimeout(function () {
          if (cursor) cursor.style.display = "none";
          if (out) { out.hidden = false; }
        }, 380);
      }
    }, 42);
  }

  if (console_) {
    if (reduced) { finalState(); }
    else if ("IntersectionObserver" in window) {
      var seen = false;
      var hio = new IntersectionObserver(function (entries) {
        if (entries[0].isIntersecting && !seen) { seen = true; setTimeout(animate, 350); hio.disconnect(); }
      }, { threshold: 0.4 });
      hio.observe(console_);
    } else { finalState(); }
  }

  /* ---- pipeline rail: interactive hover on workflow phases ---- */
  document.querySelectorAll('[data-rail="static"] .node').forEach(function (node, idx) {
    node.tabIndex = 0;
  });

  /* ---- version badge: reflect the latest published release (falls back to hardcoded) ---- */
  var versionEls = document.querySelectorAll(".js-version");
  if (versionEls.length && "fetch" in window) {
    fetch("https://api.github.com/repos/ViktorsBaikers/DevRites/releases/latest", {
      headers: { Accept: "application/vnd.github+json" },
    })
      .then(function (r) { return r.ok ? r.json() : null; })
      .then(function (d) {
        if (d && d.tag_name) versionEls.forEach(function (el) { el.textContent = d.tag_name; });
      })
      .catch(function () { /* keep the hardcoded fallback */ });
  }
})();
