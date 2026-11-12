/* Handbook viewer: search, diagram zoom/pan/full-screen, lazy Mermaid.
   No network access: the search index is a script, because file:// blocks XHR. */
(function () {
  'use strict';
  var root = document.body.getAttribute('data-root') || '';

  /* ---- search ---- */
  var input = document.getElementById('hb-q');
  var list = document.getElementById('hb-results');
  var index = window.HB_SEARCH || [];
  function terms(q) { return q.toLowerCase().split(/\s+/).filter(Boolean); }
  function snippet(text, t) {
    var i = text.toLowerCase().indexOf(t);
    if (i < 0) { return text.slice(0, 100); }
    return (i > 40 ? '...' : '') + text.slice(Math.max(0, i - 40), i + 80);
  }
  function search(ts) {
    var hits = [];
    index.forEach(function (e) {
      var title = e.title.toLowerCase(), body = e.text.toLowerCase(), score = 0;
      for (var k = 0; k < ts.length; k++) {
        var inTitle = title.indexOf(ts[k]) >= 0, inBody = body.indexOf(ts[k]) >= 0;
        if (!inTitle && !inBody) { return; }
        score += (inTitle ? 5 : 0) + (inBody ? 1 : 0);
      }
      hits.push({ e: e, score: score });
    });
    hits.sort(function (a, b) { return b.score - a.score || a.e.title.localeCompare(b.e.title); });
    return hits.slice(0, 20);
  }
  if (input && list) {
    input.addEventListener('input', function () {
      var ts = terms(input.value);
      list.textContent = '';
      if (!ts.length) { list.hidden = true; return; }
      var hits = search(ts);
      list.hidden = false;
      if (!hits.length) {
        var none = document.createElement('li');
        none.className = 'hb-none';
        none.textContent = 'No match';
        list.appendChild(none);
        return;
      }
      hits.forEach(function (h) {
        var li = document.createElement('li'), a = document.createElement('a'), s = document.createElement('small');
        a.href = root + h.e.url;
        a.textContent = h.e.title;
        s.textContent = h.e.section + ': ' + snippet(h.e.text, ts[0]);
        a.appendChild(s);
        li.appendChild(a);
        list.appendChild(li);
      });
    });
  }

  /* ---- diagrams: zoom, pan, reset, full-screen ---- */
  function enhance(fig) {
    var wrap = fig.querySelector('.dz-wrap');
    if (!wrap || wrap.querySelector('.dz-inner')) { return; }
    var inner = document.createElement('div');
    inner.className = 'dz-inner';
    Array.prototype.slice.call(wrap.childNodes).forEach(function (k) {
      if (!(k.nodeType === 1 && k.classList.contains('mermaid-src'))) { inner.appendChild(k); }
    });
    wrap.appendChild(inner);
    var st = { x: 0, y: 0, k: 1 }, over = false, drag = null;
    function apply() { inner.style.transform = 'translate(' + st.x + 'px,' + st.y + 'px) scale(' + st.k + ')'; }
    function zoom(f, cx, cy) {
      var nk = Math.min(8, Math.max(0.2, st.k * f)), r = nk / st.k;
      st.x = cx - (cx - st.x) * r; st.y = cy - (cy - st.y) * r; st.k = nk; apply();
    }
    function reset() { st.x = 0; st.y = 0; st.k = 1; apply(); }
    wrap.addEventListener('pointerenter', function () { over = true; });
    wrap.addEventListener('pointerleave', function () { over = false; });
    /* The wheel zooms only while the pointer is over the diagram, so the page still scrolls elsewhere. */
    wrap.addEventListener('wheel', function (ev) {
      if (!over) { return; }
      ev.preventDefault();
      var b = wrap.getBoundingClientRect();
      zoom(ev.deltaY < 0 ? 1.15 : 1 / 1.15, ev.clientX - b.left, ev.clientY - b.top);
    }, { passive: false });
    wrap.addEventListener('pointerdown', function (ev) {
      if (ev.button !== 0) { return; }
      drag = { x: ev.clientX - st.x, y: ev.clientY - st.y };
      wrap.classList.add('grabbing');
      wrap.setPointerCapture(ev.pointerId);
    });
    wrap.addEventListener('pointermove', function (ev) {
      if (!drag) { return; }
      st.x = ev.clientX - drag.x; st.y = ev.clientY - drag.y; apply();
    });
    function end() { drag = null; wrap.classList.remove('grabbing'); }
    wrap.addEventListener('pointerup', end);
    wrap.addEventListener('pointercancel', end);

    var bar = document.createElement('div');
    bar.className = 'dz-bar';
    function btn(label, title, fn) {
      var b = document.createElement('button');
      b.type = 'button'; b.textContent = label; b.title = title; b.setAttribute('aria-label', title);
      b.addEventListener('click', fn);
      bar.appendChild(b);
    }
    function centre() { var b = wrap.getBoundingClientRect(); return [b.width / 2, b.height / 2]; }
    btn('+', 'Zoom in', function () { var c = centre(); zoom(1.25, c[0], c[1]); });
    btn('-', 'Zoom out', function () { var c = centre(); zoom(1 / 1.25, c[0], c[1]); });
    btn('Reset', 'Reset zoom and position', reset);
    btn('Full screen', 'Full screen', function () {
      /* Fullscreen API first; where the browser refuses it (file:// policy, embedded views), fall back to a
         fixed full-window overlay so the control always works. */
      if (document.fullscreenElement === fig) { document.exitFullscreen(); return; }
      if (fig.classList.contains('dz-max')) { fig.classList.remove('dz-max'); reset(); return; }
      function overlay() { fig.classList.add('dz-max'); reset(); }
      if (!fig.requestFullscreen) { overlay(); return; }
      try {
        var p = fig.requestFullscreen();
        if (p && p.catch) { p.catch(overlay); }
      } catch (e) { overlay(); }
    });
    document.addEventListener('keydown', function (ev) {
      if (ev.key === 'Escape' && fig.classList.contains('dz-max')) { fig.classList.remove('dz-max'); reset(); }
    });
    fig.insertBefore(bar, wrap);
    document.addEventListener('fullscreenchange', reset);
  }

  /* ---- Mermaid, rendered lazily when the diagram scrolls into view ---- */
  var seq = 0;
  function renderMermaid(fig) {
    var src = fig.querySelector('pre.mermaid-src');
    if (!src || fig.getAttribute('data-done')) { return; }
    fig.setAttribute('data-done', '1');
    if (!window.mermaid) { return; }
    window.mermaid.render('hb-mmd-' + (++seq), src.textContent).then(function (res) {
      var wrap = fig.querySelector('.dz-wrap'), holder = document.createElement('div');
      holder.innerHTML = res.svg; /* sanitised by Mermaid with securityLevel 'strict' */
      while (holder.firstChild) { wrap.appendChild(holder.firstChild); }
      fig.classList.add('rendered');
      enhance(fig);
    }, function (err) {
      var e = document.createElement('div');
      e.className = 'dz-error';
      e.textContent = 'Diagram failed to render: ' + (err && err.message ? err.message : err);
      fig.appendChild(e);
    });
  }
  if (window.mermaid) {
    window.mermaid.initialize({ startOnLoad: false, securityLevel: 'strict', theme: 'neutral' });
  }
  var figs = document.querySelectorAll('figure.diagram');
  Array.prototype.forEach.call(figs, function (fig) {
    if (fig.getAttribute('data-kind') === 'excalidraw') { enhance(fig); }
  });
  var mm = Array.prototype.filter.call(figs, function (f) { return f.getAttribute('data-kind') === 'mermaid'; });
  if ('IntersectionObserver' in window) {
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (en) { if (en.isIntersecting) { io.unobserve(en.target); renderMermaid(en.target); } });
    }, { rootMargin: '200px' });
    mm.forEach(function (f) { io.observe(f); });
  } else {
    mm.forEach(renderMermaid);
  }
})();
