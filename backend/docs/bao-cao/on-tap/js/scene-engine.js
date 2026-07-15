/* Engine hoạt cảnh dùng chung.
 *
 * Mỗi hoạt cảnh khai báo:
 *   - controls: các nút / dropdown "gây lỗi" — đổi giá trị thì dựng lại kịch bản
 *   - build(opts) -> [{ view, note }]  : mảng bước, mỗi bước là HTML tĩnh
 *
 * Engine lo phần chung: khung, thanh điều khiển, bấm tới/lui, đếm bước.
 * Không có animation theo thời gian — người xem tự bấm từng bước, nên lúc bảo vệ
 * có thể dừng ở đúng khung hình cần giải thích.
 */
window.VF = window.VF || {};
VF.scenes = {};

VF.scene = function (id, def) { VF.scenes[id] = def; };

/* ── Nguyên liệu vẽ dùng chung ─────────────────────────────── */
VF.esc = function (s) {
  return String(s).replace(/[&<>"]/g, function (c) {
    return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c];
  });
};

/* Sơ đồ tuần tự vẽ bằng SVG — mỗi service là một ĐƯỜNG ĐỜI dọc, mỗi lời gọi là
 * một mũi tên bắc ngang giữa hai đường đời. Đây là ký pháp chuẩn của sequence
 * diagram: có đường đời thì mắt mới bám được "ai đang nói với ai", còn danh sách
 * mũi tên rời rạc thì không.
 *
 * rows: [{ from, to, label, cls }] — chỉ số cột; cls ∈ ok|fail|comp|wait|dead
 * now:  chỉ số hàng đang chạy (hàng sau bị làm mờ, hàng này được tô đậm)
 *
 * Bù trừ (comp) vẽ NÉT ĐỨT màu mực dấu — nhìn là biết ngay đây là đường lùi. */
VF.lanes = function (lanes, rows, now) {
  var n = lanes.length;
  var W = 880, PAD = 16, HEAD = 56, ROW = 48, TOP = HEAD + 26;
  var H = TOP + rows.length * ROW + 14;
  var colW = (W - 2 * PAD) / n;
  var x = function (i) { return PAD + colW * (i + 0.5); };

  var s = '<svg class="seq" viewBox="0 0 ' + W + ' ' + H + '" role="img" ' +
          'preserveAspectRatio="xMidYMid meet">';

  // Đường đời — kẻ trước để mũi tên nằm đè lên
  lanes.forEach(function (l, i) {
    s += '<line class="seq-life" x1="' + x(i) + '" y1="' + (HEAD + 2) + '" x2="' + x(i) + '" y2="' + (H - 8) + '"/>';
  });

  // Hộp tên service
  lanes.forEach(function (l, i) {
    var bw = Math.min(colW - 16, 190);
    s += '<g class="seq-actor">' +
         '<rect x="' + (x(i) - bw / 2) + '" y="8" width="' + bw + '" height="42" rx="2"/>' +
         '<text class="seq-actor-name" x="' + x(i) + '" y="26">' + VF.esc(l.name) + '</text>' +
         '<text class="seq-actor-sub" x="' + x(i) + '" y="41">' + VF.esc(l.sub || '') + '</text>' +
         '</g>';
  });

  rows.forEach(function (r, i) {
    var y = TOP + i * ROW;
    var cls = 'seq-msg ' + (r.cls || 'ok') +
              (i > now ? ' future' : '') + (i === now ? ' now' : '');
    var x1 = x(r.from), x2 = x(r.to);
    s += '<g class="' + cls + '">';

    if (i === now) {
      s += '<rect class="seq-now-band" x="' + PAD + '" y="' + (y - 20) + '" ' +
           'width="' + (W - 2 * PAD) + '" height="' + (ROW - 6) + '"/>';
    }

    if (r.from === r.to) {
      // Lời gọi tự thân: cung vuông quay lại chính mình (retry, chờ, crash)
      var w = 34;
      s += '<path class="seq-line" d="M' + x1 + ' ' + (y - 7) + ' h' + w +
           ' v14 h-' + (w - 8) + '" fill="none"/>' +
           arrowHead(x1 + 9, y + 7, -1) +
           // style thắng được CSS .seq-label{text-anchor:middle}; thuộc tính thường thì không
           '<text class="seq-label" style="text-anchor:start" x="' + (x1 + w + 12) + '" y="' + (y + 4) + '">' +
           VF.esc(r.label) + '</text>';
    } else {
      var dir = x2 > x1 ? 1 : -1;
      var tip = x2 - dir * 7;
      s += '<line class="seq-line" x1="' + x1 + '" y1="' + y + '" x2="' + tip + '" y2="' + y + '"/>' +
           '<circle class="seq-dot" cx="' + x1 + '" cy="' + y + '" r="2.5"/>' +
           arrowHead(x2, y, dir);

      var mid = (x1 + x2) / 2;
      var tw = r.label.length * 6.1 + 14;
      s += '<rect class="seq-label-bg" x="' + (mid - tw / 2) + '" y="' + (y - 17) + '" ' +
           'width="' + tw + '" height="15" />' +
           '<text class="seq-label" x="' + mid + '" y="' + (y - 5) + '">' + VF.esc(r.label) + '</text>';
    }
    s += '</g>';
  });

  return s + '</svg>';

  function arrowHead(px, py, dir) {
    var b = px - dir * 8;
    return '<path class="seq-tip" d="M' + px + ' ' + py + ' L' + b + ' ' + (py - 4.5) +
           ' L' + b + ' ' + (py + 4.5) + ' Z"/>';
  }
};

/* Ô số liệu lớn — dùng cho những con số PHẢI đập vào mắt (8 chứ không phải 16,
 * 5.055ms so với 25ms). tone ∈ ok|bad|warn|mute */
VF.stat = function (cards) {
  return '<div class="stats">' + cards.map(function (c) {
    return '<div class="stat ' + (c.tone || 'mute') + (c.hot ? ' hot' : '') + '">' +
      '<div class="stat-label">' + VF.esc(c.label) + '</div>' +
      '<div class="stat-value">' + c.value + '</div>' +
      (c.hint ? '<div class="stat-hint">' + c.hint + '</div>' : '') +
      '</div>';
  }).join('') + '</div>';
};

/* ── Khung + vòng đời một hoạt cảnh ────────────────────────── */
VF.mount = function (host, id) {
  var def = VF.scenes[id];
  if (!def) { host.innerHTML = '<p class="scene-note">Thiếu hoạt cảnh: ' + VF.esc(id) + '</p>'; return; }

  var opts = {};
  (def.controls || []).forEach(function (c) { opts[c.id] = c.value; });

  var steps = [], i = 0;

  var bar = document.createElement('div'); bar.className = 'scene-bar';
  var stage = document.createElement('div'); stage.className = 'scene-stage';
  var foot = document.createElement('div'); foot.className = 'scene-foot';

  bar.innerHTML = '<span class="name">' + VF.esc(def.title) + '</span>';

  (def.controls || []).forEach(function (c) {
    if (c.kind === 'select') {
      var wrap = document.createElement('label');
      wrap.className = 'ctl-wrap';
      wrap.append(c.label);
      var sel = document.createElement('select');
      sel.className = 'ctl';
      c.options.forEach(function (o) {
        var op = document.createElement('option');
        op.value = o[0]; op.textContent = o[1];
        if (o[0] === c.value) op.selected = true;
        sel.append(op);
      });
      sel.addEventListener('change', function () { opts[c.id] = sel.value; rebuild(); });
      wrap.append(sel);
      bar.append(wrap);
    } else {
      var b = document.createElement('button');
      b.className = 'ctl fault';
      b.textContent = c.label;
      b.setAttribute('aria-pressed', String(!!c.value));
      b.addEventListener('click', function () {
        opts[c.id] = !opts[c.id];
        b.setAttribute('aria-pressed', String(opts[c.id]));
        rebuild();
      });
      bar.append(b);
    }
  });

  var note = document.createElement('p'); note.className = 'scene-note';
  var count = document.createElement('span'); count.className = 'steps';
  var prev = document.createElement('button'); prev.className = 'ctl'; prev.textContent = '◀ Trước';
  var next = document.createElement('button'); next.className = 'ctl go'; next.textContent = 'Sau ▶';
  var again = document.createElement('button'); again.className = 'ctl'; again.textContent = '↺';
  again.title = 'Về bước đầu';
  foot.append(note, count, again, prev, next);

  prev.addEventListener('click', function () { if (i > 0) { i--; draw(); } });
  next.addEventListener('click', function () { if (i < steps.length - 1) { i++; draw(); } });
  again.addEventListener('click', function () { i = 0; draw(); });

  function draw() {
    stage.innerHTML = steps[i].view;
    note.innerHTML = steps[i].note || '';
    count.textContent = 'Bước ' + (i + 1) + '/' + steps.length;
    prev.disabled = i === 0;
    next.disabled = i === steps.length - 1;
  }
  function rebuild() { steps = def.build(opts); i = 0; draw(); }

  host.classList.add('scene');
  host.append(bar, stage, foot);
  rebuild();
};

VF.mountAll = function (root) {
  root.querySelectorAll('[data-scene]').forEach(function (el) {
    if (el.dataset.mounted) return;
    el.dataset.mounted = '1';
    VF.mount(el, el.dataset.scene);
  });
};
