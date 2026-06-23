/* DevRites landing — cinematic control surface.
   Motion (vendored framer-motion/dom) drives reveals, scroll-linking & power-on.
   A WebGL blade-mesh lights the hero. All of it is progressive enhancement:
   without JS the content is fully visible, and prefers-reduced-motion renders
   every final state instantly with no animation, no shader. */

const root = document.documentElement;
root.classList.add("js");
const reduced = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
const $ = (s, c = document) => c.querySelector(s);
const $$ = (s, c = document) => Array.from(c.querySelectorAll(s));

/* ============================================================ *
 *  Baseline UX — never depends on Motion or WebGL loading.     *
 * ============================================================ */

/* nav: sticky glass shadow + mobile toggle */
(function nav() {
  const nav = $("#nav");
  if (!nav) return;
  const onScroll = () => nav.classList.toggle("is-stuck", window.scrollY > 8);
  onScroll();
  window.addEventListener("scroll", onScroll, { passive: true });

  const burger = $(".nav__burger", nav);
  if (!burger) return;
  const close = () => { nav.classList.remove("is-open"); burger.setAttribute("aria-expanded", "false"); };
  burger.addEventListener("click", () => {
    const open = nav.classList.toggle("is-open");
    burger.setAttribute("aria-expanded", open ? "true" : "false");
  });
  $$(".nav__links a", nav).forEach((a) => a.addEventListener("click", close));
  document.addEventListener("keydown", (e) => { if (e.key === "Escape") close(); });
})();

/* copy buttons (data-copy wins, else #installCmd text) */
(function copy() {
  if (!navigator.clipboard) return;
  $$(".copybtn").forEach((btn) => {
    const label = $("span", btn);
    btn.addEventListener("click", () => {
      let text = btn.getAttribute("data-copy");
      if (!text) { const cmd = $("#installCmd"); text = cmd ? cmd.innerText.trim() : ""; }
      if (!text) return;
      navigator.clipboard.writeText(text).then(() => {
        btn.classList.add("copied");
        const prev = label ? label.textContent : "";
        if (label) label.textContent = "copied";
        setTimeout(() => { btn.classList.remove("copied"); if (label) label.textContent = prev; }, 1800);
      });
    });
  });
})();

/* version badge — latest published release, falls back to hardcoded */
(function version() {
  const els = $$(".js-version");
  if (!els.length || !("fetch" in window)) return;
  fetch("https://api.github.com/repos/ViktorsBaikers/DevRites/releases/latest", {
    headers: { Accept: "application/vnd.github+json" },
  })
    .then((r) => (r.ok ? r.json() : null))
    .then((d) => { if (d && d.tag_name) els.forEach((el) => (el.textContent = d.tag_name)); })
    .catch(() => {});
})();

/* social proof — live GitHub star count, shown only when it's worth showing */
(function stars() {
  const pill = $("#starPill"), num = $("#starCount");
  if (!pill || !num || !("fetch" in window)) return;
  fetch("https://api.github.com/repos/ViktorsBaikers/DevRites", { headers: { Accept: "application/vnd.github+json" } })
    .then((r) => (r.ok ? r.json() : null))
    .then((d) => {
      const n = d && d.stargazers_count;
      if (!n || n < 10) return; // never show an embarrassing or fabricated number
      num.textContent = n >= 1000 ? (n / 1000).toFixed(1).replace(/\.0$/, "") + "k" : String(n);
      pill.hidden = false;
    })
    .catch(() => {});
})();

/* workflow rail nodes are focusable for keyboard inspection */
$$('[data-rail="static"] .node').forEach((n) => (n.tabIndex = 0));

/* console power-on geometry (shared by Motion path and reduced/fallback path) */
const consoleEl = $("#console");
const railFill = $("#railFill");
const nodes = consoleEl ? $$(".node", consoleEl) : [];
const CURRENT = 4; // build = index 4 of spec·temper·define·vet·build…
const railTarget = nodes.length > 1 ? CURRENT / (nodes.length - 1) : 0;
const cmdEl = $("#termCmd");
const cursor = $("#termCursor");
const out = $("#termOut");
const FULL = "/rite-build refresh-token";

// reveal the result block; CSS staggers its lines in (instant under reduced-motion)
function revealOut() {
  if (!out) return;
  out.hidden = false;
  requestAnimationFrame(() => out.classList.add("is-revealed"));
}

function consoleFinal() {
  nodes.forEach((n, i) => {
    n.classList.remove("is-done", "is-current");
    if (i < CURRENT) n.classList.add("is-done");
    else if (i === CURRENT) n.classList.add("is-current");
  });
  if (railFill) railFill.style.transform = `scaleX(${railTarget})`;
  if (cmdEl) cmdEl.textContent = FULL;
  if (cursor) cursor.style.display = "none";
  revealOut();
}

function typeCommand(done) {
  let i = 0;
  const t = setInterval(() => {
    if (cmdEl) cmdEl.textContent = FULL.slice(0, ++i);
    if (i >= FULL.length) {
      clearInterval(t);
      setTimeout(() => { if (cursor) cursor.style.display = "none"; revealOut(); done && done(); }, 360);
    }
  }, 42);
}

/* ============================================================ *
 *  Reduced motion — final states, no animation, no shader.     *
 * ============================================================ */
if (reduced) {
  $$(".reveal").forEach((el) => el.classList.add("in"));
  const wf = $("#workflowFill"); if (wf) wf.style.transform = "scaleX(1)";
  const sf = $("#scrollFill"); if (sf) sf.style.transform = "scaleX(1)";
  consoleFinal();
}

/* ============================================================ *
 *  Motion + shader — loaded dynamically, degrade gracefully.   *
 * ============================================================ */
if (!reduced) {
  // Fallback reveal via IntersectionObserver if Motion fails to load.
  let revealed = false;
  const ioReveal = () => {
    if (revealed) return; revealed = true;
    if (!("IntersectionObserver" in window)) { $$(".reveal").forEach((el) => el.classList.add("in")); return; }
    const io = new IntersectionObserver((ents) => {
      ents.forEach((en) => { if (en.isIntersecting) { en.target.classList.add("in"); io.unobserve(en.target); } });
    }, { threshold: 0.12, rootMargin: "0px 0px -8% 0px" });
    $$(".reveal").forEach((el) => io.observe(el));
  };

  import("/assets/js/vendor/motion.min.js")
    .then((M) => initMotion(M))
    .catch(() => {
      // Motion unavailable: still reveal everything and show final console.
      ioReveal();
      const wf = $("#workflowFill"); if (wf) wf.style.transform = "scaleX(1)";
      startConsoleFallback();
    });

  // shader is independent of Motion — start it regardless.
  startShader();

  function initMotion({ animate, inView, scroll }) {
    const EASE = [0.16, 1, 0.3, 1];

    /* scroll-driven reveals — opacity + lift, honoring per-element data-d delay */
    const seen = new WeakSet();
    const revealNow = (el) => {
      if (!el || seen.has(el)) return; seen.add(el);
      const d = parseFloat(el.getAttribute("data-d") || "0") * 0.08;
      animate(el, { opacity: [0, 1], transform: ["translateY(22px)", "translateY(0px)"] },
        { duration: 0.7, delay: d, ease: EASE });
    };
    inView(".reveal", (info) => revealNow(info.target || info), { amount: 0.15 });
    /* safety net: fling-scroll / End key / deep-link jumps can outrun IntersectionObserver —
       sweep (rAF-throttled) reveals anything that has entered or passed the viewport. */
    let sweepQ = false;
    const sweep = () => {
      sweepQ = false;
      $$(".reveal").forEach((el) => { if (!seen.has(el) && el.getBoundingClientRect().top < innerHeight * 0.92) revealNow(el); });
    };
    window.addEventListener("scroll", () => { if (!sweepQ) { sweepQ = true; requestAnimationFrame(sweep); } }, { passive: true });

    /* top progress bar — linked to whole-page scroll */
    const scrollFill = $("#scrollFill");
    if (scrollFill) {
      try { scroll(animate(scrollFill, { scaleX: [0, 1] }, { ease: "linear" })); }
      catch { window.addEventListener("scroll", () => {
        const p = window.scrollY / (document.body.scrollHeight - innerHeight || 1);
        scrollFill.style.transform = `scaleX(${Math.min(1, p)})`;
      }, { passive: true }); }
    }

    /* workflow rail — fills as the section scrolls through view */
    const workflowFill = $("#workflowFill");
    const workflowEl = $("#workflow");
    if (workflowFill && workflowEl) {
      try { scroll(animate(workflowFill, { scaleX: [0, 1] }, { ease: "linear" }),
        { target: workflowEl, offset: ["start end", "end center"] }); }
      catch { workflowFill.style.transform = "scaleX(1)"; }
    }

    /* hero console power-on — light nodes, grow rail, type the command */
    if (consoleEl) {
      let played = false;
      inView(consoleEl, () => {
        if (played) return; played = true;
        nodes.forEach((n, i) => {
          if (i <= CURRENT) setTimeout(() => n.classList.add(i === CURRENT ? "is-current" : "is-done"), 120 + i * 230);
        });
        const after = 120 + CURRENT * 230 + 200;
        setTimeout(() => {
          if (railFill) animate(railFill, { transform: ["scaleX(0)", `scaleX(${railTarget})`] }, { duration: 0.9, ease: EASE });
          setTimeout(() => typeCommand(), 500);
        }, after);
      }, { amount: 0.4 });
    }

    /* magnetic primary CTA — subtle spring pull toward the cursor (pointer only) */
    if (window.matchMedia("(pointer:fine)").matches) {
      $$(".hero__actions .btn--primary").forEach((btn) => {
        btn.addEventListener("pointermove", (e) => {
          const r = btn.getBoundingClientRect();
          const x = (e.clientX - r.left - r.width / 2) * 0.25;
          const y = (e.clientY - r.top - r.height / 2) * 0.35;
          animate(btn, { transform: `translate(${x}px, ${y - 2}px)` }, { duration: 0.3, ease: EASE });
        });
        btn.addEventListener("pointerleave", () => {
          animate(btn, { transform: "translate(0px,0px)" }, { type: "spring", stiffness: 250, damping: 18 });
        });
      });
    }
  }

  /* console power-on without Motion (timed CSS transitions handle the rail) */
  function startConsoleFallback() {
    if (!consoleEl) return;
    if (!("IntersectionObserver" in window)) { consoleFinal(); return; }
    let played = false;
    const io = new IntersectionObserver((ents) => {
      if (ents[0].isIntersecting && !played) {
        played = true; io.disconnect();
        nodes.forEach((n, i) => { if (i <= CURRENT) setTimeout(() => n.classList.add(i === CURRENT ? "is-current" : "is-done"), 120 + i * 230); });
        setTimeout(() => { if (railFill) railFill.style.transform = `scaleX(${railTarget})`; typeCommand(); }, 120 + CURRENT * 230 + 300);
      }
    }, { threshold: 0.4 });
    io.observe(consoleEl);
  }
}

/* ============================================================ *
 *  WebGL blade-mesh shader — flowing navy → cyan → blue field. *
 *  Cheap (5-octave value-noise fbm), DPR-capped, pauses when   *
 *  off-screen or the tab is hidden. CSS gradient is the floor. *
 * ============================================================ */
function startShader() {
  const canvas = $("#shader");
  if (!canvas) return;
  let gl;
  try { gl = canvas.getContext("webgl", { antialias: false, alpha: true, premultipliedAlpha: false }); }
  catch { return; }
  if (!gl) return; // CSS gradient stays as the visual floor

  const VERT = "attribute vec2 p;void main(){gl_Position=vec4(p,0.0,1.0);}";
  const FRAG = `precision highp float;
uniform vec2 r;uniform float t;
float hash(vec2 p){return fract(sin(dot(p,vec2(127.1,311.7)))*43758.5453);}
float noise(vec2 p){vec2 i=floor(p),f=fract(p);f=f*f*(3.0-2.0*f);
 float a=hash(i),b=hash(i+vec2(1.0,0.0)),c=hash(i+vec2(0.0,1.0)),d=hash(i+vec2(1.0,1.0));
 return mix(mix(a,b,f.x),mix(c,d,f.x),f.y);}
float fbm(vec2 p){float v=0.0,a=0.5;for(int i=0;i<5;i++){v+=a*noise(p);p*=2.02;a*=0.5;}return v;}
void main(){
 vec2 uv=gl_FragCoord.xy/r.xy;
 vec2 q=uv;q.x*=r.x/r.y;
 float n=fbm(q*1.7+vec2(t*0.030,t*0.020));
 float m=fbm(q*2.4-vec2(t*0.022,t*0.026)+n*0.9);
 vec3 navy=vec3(0.050,0.078,0.190);
 vec3 cyan=vec3(0.330,0.820,0.900);
 vec3 blue=vec3(0.230,0.420,0.930);
 vec3 blade=mix(blue,cyan,smoothstep(0.20,0.85,m));
 float glow=smoothstep(0.50,0.96,m)*0.85+smoothstep(0.0,0.45,n)*0.12;
 vec3 col=mix(navy,blade,clamp(glow,0.0,1.0));
 float vig=smoothstep(1.20,0.18,length(uv-vec2(0.55,0.32)));
 col*=vig;
 gl_FragColor=vec4(col,1.0);
}`;

  function compile(type, src) {
    const s = gl.createShader(type); gl.shaderSource(s, src); gl.compileShader(s);
    if (!gl.getShaderParameter(s, gl.COMPILE_STATUS)) return null;
    return s;
  }
  const vs = compile(gl.VERTEX_SHADER, VERT);
  const fs = compile(gl.FRAGMENT_SHADER, FRAG);
  if (!vs || !fs) return;
  const prog = gl.createProgram();
  gl.attachShader(prog, vs); gl.attachShader(prog, fs); gl.linkProgram(prog);
  if (!gl.getProgramParameter(prog, gl.LINK_STATUS)) return;
  gl.useProgram(prog);

  const buf = gl.createBuffer();
  gl.bindBuffer(gl.ARRAY_BUFFER, buf);
  gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 3, -1, -1, 3]), gl.STATIC_DRAW);
  const loc = gl.getAttribLocation(prog, "p");
  gl.enableVertexAttribArray(loc);
  gl.vertexAttribPointer(loc, 2, gl.FLOAT, false, 0, 0);
  const uR = gl.getUniformLocation(prog, "r");
  const uT = gl.getUniformLocation(prog, "t");

  const DPR = Math.min(window.devicePixelRatio || 1, 1.5);
  function resize() {
    const w = Math.max(1, Math.floor(canvas.clientWidth * DPR));
    const h = Math.max(1, Math.floor(canvas.clientHeight * DPR));
    if (canvas.width !== w || canvas.height !== h) { canvas.width = w; canvas.height = h; gl.viewport(0, 0, w, h); }
  }
  window.addEventListener("resize", resize, { passive: true });

  canvas.classList.add("is-on");
  let raf = 0, t0 = 0, last = 0, visible = true;

  const io = "IntersectionObserver" in window
    ? new IntersectionObserver((e) => { visible = e[0].isIntersecting; loop(performance.now()); }, { threshold: 0 })
    : null;
  if (io) io.observe(canvas);
  document.addEventListener("visibilitychange", () => { if (!document.hidden) loop(performance.now()); });

  function loop(now) {
    cancelAnimationFrame(raf);
    if (!visible || document.hidden) return; // pause when off-screen / tab hidden
    if (!t0) t0 = now;
    // throttle to ~40fps — plenty for a slow flowing field, saves battery
    if (now - last >= 24) {
      last = now;
      resize();
      gl.uniform2f(uR, canvas.width, canvas.height);
      gl.uniform1f(uT, (now - t0) / 1000);
      gl.drawArrays(gl.TRIANGLES, 0, 3);
    }
    raf = requestAnimationFrame(loop);
  }
  resize();
  raf = requestAnimationFrame(loop);
}
