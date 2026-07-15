/* Ráp trang: tab → rail mục lục → các thẻ khái niệm → gắn hoạt cảnh. */
(function () {
  var TABS = ['tongquan', 'van', 'duc', 'nam', 'hoicheo'];

  var tabsEl = document.getElementById('tabs');
  var railEl = document.getElementById('rail');
  var colEl = document.getElementById('column');
  var shell = document.getElementById('shell');

  function cardHTML(c) {
    var h = '<section class="card" id="' + c.id + '"><header>';
    if (c.eyebrow && c.eyebrow.length) {
      h += '<div class="eyebrow">' + c.eyebrow.map(function (e, i) {
        return i === 0 ? '<span class="req">' + e + '</span>' : '<span>' + e + '</span>';
      }).join('') + '</div>';
    }
    h += '<h2>' + c.title + '</h2>';
    if (c.lede) h += '<p class="lede">' + c.lede + '</p>';
    h += '</header>';

    if (c.scene) h += '<div data-scene="' + c.scene + '"></div>';
    if (c.html) h += '<div class="freeform">' + c.html + '</div>';

    if (c.say && c.say.length) {
      h += '<div class="say"><span class="label">Câu phải nói</span>' +
           c.say.map(function (p) { return '<p>' + p + '</p>'; }).join('') + '</div>';
    }

    if (c.probes && c.probes.length) {
      h += '<div class="probe"><span class="label">Nếu thầy hỏi sâu</span>';
      c.probes.forEach(function (p) {
        h += '<details><summary>' + p[0] + '</summary><div class="a">' + p[1] + '</div></details>';
      });
      h += '</div>';
    }

    if (c.srcs && c.srcs.length) {
      h += '<div class="srcs">' + c.srcs.map(function (s) {
        return '<span class="src">' + VF.esc(s) + '</span>';
      }).join('') + '</div>';
    }

    return h + '</section>';
  }

  function render(key) {
    var data = VF.content[key];
    shell.style.setProperty('--tab-accent', data.accent);

    railEl.innerHTML = '<div class="rail-title">' + VF.esc(data.label) + '</div><ol>' +
      data.cards.map(function (c, i) {
        return '<li><a href="#' + c.id + '"' + (c.scene ? ' class="has-scene"' : '') + '>' +
               '<span class="n">' + (i + 1) + '</span><span>' + stripTags(c.title) + '</span></a></li>';
      }).join('') + '</ol>';

    colEl.innerHTML = '<p class="lede" style="font-size:18px;border-bottom:1px solid var(--rule);padding-bottom:24px;margin:0">' +
      data.blurb + '</p>' + data.cards.map(cardHTML).join('');

    VF.mountAll(colEl);
    spy();
    window.scrollTo(0, 0);
  }

  function stripTags(s) { return s.replace(/<[^>]+>/g, ''); }

  /* Tô sáng mục đang đọc trong rail */
  function spy() {
    var links = Array.prototype.slice.call(railEl.querySelectorAll('a'));
    var cards = links.map(function (a) { return document.querySelector(a.getAttribute('href')); });
    function update() {
      var best = 0;
      cards.forEach(function (c, i) {
        if (c && c.getBoundingClientRect().top < 160) best = i;
      });
      links.forEach(function (a, i) { a.classList.toggle('on', i === best); });
    }
    window.removeEventListener('scroll', window.__vfSpy);
    window.__vfSpy = update;
    window.addEventListener('scroll', update, { passive: true });
    update();
  }

  TABS.forEach(function (key, i) {
    var d = VF.content[key];
    var b = document.createElement('button');
    b.className = 'tab';
    b.setAttribute('role', 'tab');
    b.setAttribute('aria-selected', String(i === 0));
    b.innerHTML = VF.esc(d.label) + (d.tally ? '<span class="count">' + d.tally + '</span>' : '');
    b.addEventListener('click', function () {
      tabsEl.querySelectorAll('.tab').forEach(function (t) { t.setAttribute('aria-selected', 'false'); });
      b.setAttribute('aria-selected', 'true');
      location.hash = '#tab=' + key;
      render(key);
    });
    tabsEl.append(b);
  });

  var start = (location.hash.match(/^#tab=(\w+)$/) || [])[1];
  if (TABS.indexOf(start) < 0) start = 'tongquan';
  tabsEl.querySelectorAll('.tab')[TABS.indexOf(start)].setAttribute('aria-selected', 'true');
  tabsEl.querySelectorAll('.tab')[0].setAttribute('aria-selected', String(start === 'tongquan'));
  render(start);
})();
