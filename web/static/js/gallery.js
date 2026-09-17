(function () {
  var INTERVAL = 6000;

  function parseJSON(el) {
    if (!el) return [];
    try { return JSON.parse(el.textContent); } catch (e) { return []; }
  }

  function pageMath(n, p, total) {
    var per = (n === 12 || n === 18 || n === 24) ? n : 6;
    if (total <= 0) return { per: per, page: 1, from: 0, to: 0, offset: 0 };
    var pages = Math.ceil(total / per);
    var page = p < 1 ? 1 : p;
    if (page > pages) page = pages;
    var offset = (page - 1) * per;
    var to = offset + per;
    if (to > total) to = total;
    return { per: per, page: page, from: offset + 1, to: to, offset: offset };
  }

  function init(root) {
    var photos = parseJSON(document.querySelector(root.getAttribute('data-json')));
    if (!photos.length) return;
    var img = root.querySelector('.slideshow-img');
    var strip = root.querySelector('.slideshow-strip');
    var prev = root.querySelector('.slideshow-prev');
    var next = root.querySelector('.slideshow-next');
    var tprev = root.querySelector('.slideshow-tprev');
    var tnext = root.querySelector('.slideshow-tnext');
    var dlg = root.querySelector('.slideshow-full');
    var full = root.querySelector('.slideshow-full-img');
    var fprev = root.querySelector('.slideshow-full-prev');
    var fnext = root.querySelector('.slideshow-full-next');
    var fclose = root.querySelector('.slideshow-full-close');
    var credits = root.querySelectorAll('.photo-credit');
    var i = 0;
    var win = 0;
    var timer;

    function credit(ph) {
      for (var c = 0; c < credits.length; c++) {
        var el = credits[c];
        if (!ph.name) { el.hidden = true; continue; }
        el.hidden = false;
        el.textContent = ph.name;
        if (ph.href) el.setAttribute('href', ph.href);
        else el.removeAttribute('href');
      }
    }

    function showThumbs() {
      if (!strip || !tprev || !tnext) return;
      var n = photos.length;
      if (n <= 3) {
        win = 0;
        tprev.hidden = true;
        tnext.hidden = true;
      } else {
        tprev.hidden = false;
        tnext.hidden = false;
        if (win > n - 3) win = n - 3;
        if (win < 0) win = 0;
      }
      strip.innerHTML = '';
      var end = Math.min(win + 3, n);
      for (var t = win; t < end; t++) {
        var b = document.createElement('button');
        b.type = 'button';
        b.className = 'slideshow-thumb' + (t === i ? ' is-on' : '');
        b.setAttribute('aria-label', 'Photo ' + (t + 1));
        if (t === i) b.setAttribute('aria-current', 'true');
        var th = document.createElement('img');
        th.src = photos[t].url;
        th.alt = '';
        b.appendChild(th);
        b.addEventListener('click', (function (idx) {
          return function () { go(idx); };
        })(t));
        strip.appendChild(b);
      }
    }

    function paint() {
      img.src = photos[i].url;
      img.alt = photos[i].name || '';
      prev.hidden = photos.length < 2;
      next.hidden = photos.length < 2;
      if (fprev) fprev.hidden = photos.length < 2;
      if (fnext) fnext.hidden = photos.length < 2;
      if (full) {
        full.src = photos[i].url;
        full.alt = photos[i].name || '';
      }
      credit(photos[i]);
      if (i < win || i >= win + 3) win = Math.floor(i / 3) * 3;
      showThumbs();
    }

    function tick() {
      i = (i + 1) % photos.length;
      paint();
    }

    function arm() {
      clearInterval(timer);
      if (photos.length > 1) timer = setInterval(tick, INTERVAL);
    }

    function go(idx) {
      i = (idx + photos.length) % photos.length;
      paint();
      arm();
    }

    prev.addEventListener('click', function () { go(i - 1); });
    next.addEventListener('click', function () { go(i + 1); });
    img.addEventListener('click', function () {
      if (!dlg || !dlg.showModal) return;
      paint();
      dlg.showModal();
      arm();
    });
    credits.forEach(function (el) {
      el.addEventListener('click', function (e) { e.stopPropagation(); });
    });
    if (fprev) fprev.addEventListener('click', function (e) { e.stopPropagation(); go(i - 1); });
    if (fnext) fnext.addEventListener('click', function (e) { e.stopPropagation(); go(i + 1); });
    if (fclose) fclose.addEventListener('click', function () { dlg.close(); });
    if (dlg) {
      dlg.addEventListener('click', function (e) { if (e.target === dlg) dlg.close(); });
      dlg.addEventListener('close', arm);
      dlg.addEventListener('keydown', function (e) {
        if (e.key === 'ArrowLeft') { e.preventDefault(); go(i - 1); }
        if (e.key === 'ArrowRight') { e.preventDefault(); go(i + 1); }
      });
      var tx = 0, ty = 0;
      dlg.addEventListener('touchstart', function (e) {
        var t = e.changedTouches[0];
        tx = t.clientX;
        ty = t.clientY;
      }, { passive: true });
      dlg.addEventListener('touchend', function (e) {
        var t = e.changedTouches[0];
        var dx = t.clientX - tx, dy = t.clientY - ty;
        if (Math.abs(dx) < 40 && Math.abs(dy) < 40) return;
        if (Math.abs(dy) > Math.abs(dx)) {
          if (dy > 40) dlg.close();
          return;
        }
        go(i + (dx < 0 ? 1 : -1));
      });
    }
    if (tprev) tprev.addEventListener('click', function () {
      win -= 3;
      if (win < 0) win = Math.max(0, photos.length - 3);
      showThumbs();
    });
    if (tnext) tnext.addEventListener('click', function () {
      win += 3;
      if (win >= photos.length) win = 0;
      showThumbs();
    });

    paint();
    arm();

    if (root.getAttribute('data-live') !== '1') return;

    var grid = document.querySelector(root.getAttribute('data-grid'));
    var pager = document.querySelector(root.getAttribute('data-pager'));
    var params = new URLSearchParams(location.search);
    var n = parseInt(params.get('n') || '6', 10) || 6;
    var p = parseInt(params.get('p') || '1', 10) || 1;

    function nav(on, href, label, text) {
      return on
        ? '<a href="' + href + '" aria-label="' + label + '">' + text + '</a>'
        : '<span aria-disabled="true">' + text + '</span>';
    }

    function render() {
      var m = pageMath(n, p, photos.length);
      n = m.per;
      p = m.page;
      if (grid) {
        grid.innerHTML = '';
        photos.slice(m.offset, m.to).forEach(function (ph, j) {
          var li = document.createElement('li');
          li.className = 'photo-cell';
          var a = document.createElement('a');
          a.href = '#';
          a.setAttribute('data-idx', String(m.offset + j));
          var im = document.createElement('img');
          im.src = ph.url;
          im.alt = '';
          im.className = 'thumb';
          a.appendChild(im);
          li.appendChild(a);
          if (ph.name) {
            var who = document.createElement(ph.href ? 'a' : 'span');
            who.className = 'photo-credit';
            who.textContent = ph.name;
            if (ph.href) who.href = ph.href;
            li.appendChild(who);
          }
          grid.appendChild(li);
        });
      }
      if (pager) {
        var sizes = [6, 12, 18, 24];
        var sizeLinks = sizes.map(function (s) {
          return '<a href="/photos?n=' + s + '"' + (s === m.per ? ' aria-current="true"' : '') + '>' + s + '</a>';
        }).join('');
        pager.innerHTML =
          '<span class="pager-size">Rows per page:<details><summary>' + m.per + '</summary><div>' + sizeLinks + '</div></details></span>' +
          '<span class="pager-range">' + m.from + '–' + m.to + ' of ' + photos.length + '</span>' +
          nav(m.page > 1, '/photos?n=' + m.per + '&p=1', 'First page', '|‹') +
          nav(m.page > 1, '/photos?n=' + m.per + '&p=' + (m.page - 1), 'Previous page', '‹') +
          nav(m.to < photos.length, '/photos?n=' + m.per + '&p=' + (m.page + 1), 'Next page', '›');
      }
    }

    if (pager) {
      pager.addEventListener('click', function (e) {
        var a = e.target.closest('a');
        if (!a || !pager.contains(a)) return;
        e.preventDefault();
        var u = new URL(a.href, location.origin);
        n = parseInt(u.searchParams.get('n') || String(n), 10);
        p = parseInt(u.searchParams.get('p') || '1', 10);
        render();
        history.replaceState({ n: n, p: p }, '', '/photos?n=' + n + '&p=' + p);
      });
    }
    if (grid) {
      grid.addEventListener('click', function (e) {
        var a = e.target.closest('a[data-idx]');
        if (!a) return;
        e.preventDefault();
        go(parseInt(a.getAttribute('data-idx'), 10));
      });
    }
    render();
  }

  document.querySelectorAll('[data-gallery]').forEach(init);
})();
