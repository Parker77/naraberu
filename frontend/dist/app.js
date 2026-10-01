// Wails v3 bindings via window.naraberuService (loaded by boot.js)
function goApi() {
    return window.naraberuService;
}

async function waitForService(timeoutMs = 10000) {
    if (window.naraberuService) return window.naraberuService;
    return new Promise((resolve, reject) => {
        const onReady = () => {
            cleanup();
            resolve(window.naraberuService);
        };
        const cleanup = () => {
            window.removeEventListener('naraberu-service-ready', onReady);
            clearInterval(poll);
        };
        window.addEventListener('naraberu-service-ready', onReady);
        const started = Date.now();
        const poll = setInterval(() => {
            if (window.naraberuService) {
                cleanup();
                resolve(window.naraberuService);
                return;
            }
            if (Date.now() - started > timeoutMs) {
                cleanup();
                reject(new Error('Wails service bindings did not load'));
            }
        }, 25);
    });
}
const GoAPI = {
    GetAllSeries: () => goApi().GetAllSeries(),
    GetSeries: (id) => goApi().GetSeries(id),
    CreateSeries: (s) => goApi().CreateSeries(s),
    UpdateSeries: (id, s) => goApi().UpdateSeries(id, s),
    DeleteSeries: (id) => goApi().DeleteSeries(id),
    SearchSeries: (f) => goApi().SearchSeries(f),
    GetAllTags: () => goApi().GetAllTags(),
    ExportCSV: () => goApi().ExportCSV(),
    ImportCSV: (d) => goApi().ImportCSV(d),
    PickThumbnail: () => goApi().PickThumbnail(),
    ReadThumbnail: (p) => goApi().ReadThumbnail(p),
    ToggleFavorite: (id) => goApi().ToggleFavorite(id),
    ToggleOwned: (id) => goApi().ToggleOwned(id),
    GetTheme: () => goApi().GetTheme(),
    SetTheme: (theme) => goApi().SetTheme(theme),
    GetAudioFile: () => goApi().GetAudioFile(),
    SetAudioFile: (file) => goApi().SetAudioFile(file),
    GetMuteAudio: () => goApi().GetMuteAudio(),
    SetMuteAudio: (mute) => goApi().SetMuteAudio(mute),
    ListAudioFiles: () => goApi().ListAudioFiles(),
    ReadAudioFile: (name) => goApi().ReadAudioFile(name),
};

let currentFilter = { query: '', status: '', tags: [], favoritesOnly: false, linkedOnly: false, ownedOnly: false, sortBy: 'updatedAt', sortDesc: false };
let allSeries = [];
let _totalSeriesCount = 0;
let allTags = [];
let currentSeriesId = null;

let searchTimeout = null;
function debouncedSearch() {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(() => {
        currentFilter.query = document.getElementById('search-input').value;
        searchSeries();
    }, 200);
}

function onSortByChanged() {
    const val = document.getElementById('sort-by').value;
    if (['updatedAt', 'score', 'startDate', 'endDate'].includes(val)) {
        document.getElementById('sort-desc').checked = true;
    } else {
        document.getElementById('sort-desc').checked = false;
    }
    searchSeries();
}

function hasActiveFilters() {
    return currentFilter.query !== '' ||
           currentFilter.status !== '' ||
           currentFilter.favoritesOnly ||
           currentFilter.linkedOnly ||
           currentFilter.ownedOnly ||
           currentFilter.tags.length > 0;
}

function updateResultCount() {
    const el = document.getElementById('result-count');
    if (!el) return;
    if (hasActiveFilters()) {
        el.textContent = allSeries.length + ' of ' + _totalSeriesCount;
    } else {
        el.textContent = _totalSeriesCount + ' series';
    }
}

async function refreshTotalCount() {
    try {
        const all = await GoAPI.GetAllSeries();
        _totalSeriesCount = (all || []).length;
    } catch (e) {}
}

async function searchSeries() {
    if (_searchGuardCooldown) return;
    if (!_searchGuardBypassed && !document.getElementById('series-detail').classList.contains('hidden') && hasUnsavedChanges()) {
        clearTimeout(searchTimeout);
        const proceed = confirm('You have unsaved changes. Leave without saving?');
        clearTimeout(searchTimeout);
        if (!proceed) {
            _searchGuardCooldown = true;
            setTimeout(() => { _searchGuardCooldown = false; }, 500);
            return;
        }
    }
    _searchGuardBypassed = false;
    currentFilter.query = document.getElementById('search-input').value;
    currentFilter.status = document.getElementById('filter-status').value;
    currentFilter.favoritesOnly = document.getElementById('filter-favorites').checked;
    currentFilter.linkedOnly = document.getElementById('filter-linked').checked;
    currentFilter.ownedOnly = document.getElementById('filter-owned').checked;
    currentFilter.sortBy = document.getElementById('sort-by').value;
    currentFilter.sortDesc = document.getElementById('sort-desc').checked;
    try {
        allSeries = await GoAPI.SearchSeries(currentFilter) || [];
        renderSeriesList();
        updateResultCount();
        updateBatchBar();
    } catch (e) { console.error('Search failed:', e); }
}

async function loadTags() {
    try {
        allTags = await GoAPI.GetAllTags();
        renderTagFilters();
    } catch (e) { console.error('Failed to load tags:', e); }
}

function renderTagFilters() {
    const container = document.getElementById('tag-filters');
    const sorted = [...allTags].sort((a, b) => a.localeCompare(b));
    container.innerHTML = sorted.map(tag => {
        const active = currentFilter.tags.some(t => t.toLowerCase() === tag.toLowerCase()) ? 'active' : '';
        return `<span class="tag-chip ${active}" onclick="toggleTagFilter('${escapeHtml(tag)}')">${escapeHtml(tag)}</span>`;
    }).join('');
    const btn = document.getElementById('tag-expand-btn');
    if (btn) btn.style.display = sorted.length > 0 ? '' : 'none';
}

function toggleTagExpand() {
    const wrapper = document.getElementById('tag-scroll-wrapper');
    const btn = document.getElementById('tag-expand-btn');
    wrapper.classList.toggle('expanded');
    if (wrapper.classList.contains('expanded')) {
        btn.textContent = 'Collapse ▴';
    } else {
        btn.textContent = 'Expand ▾';
    }
}

function toggleTagFilter(tag) {
    const idx = currentFilter.tags.findIndex(t => t.toLowerCase() === tag.toLowerCase());
    if (idx === -1) currentFilter.tags.push(tag);
    else currentFilter.tags.splice(idx, 1);
    renderTagFilters();
    searchSeries();
}

let _selectedIds = new Set();
let _thumbnailCache = {};
const _THUMBNAIL_CACHE_MAX = 100;

async function renderSeriesList() {
    const container = document.getElementById('series-list');
    const detail = document.getElementById('series-detail');
    detail.classList.add('hidden');
    container.classList.remove('hidden');
    container.style.visibility = 'hidden';

    if (allSeries.length === 0) {
        container.innerHTML = `<div class="empty-state" style="grid-column: 1 / -1"><h2>No series found</h2><p>Add your first anime series to get started.</p></div>`;
        container.style.visibility = '';
        return;
    }

    container.innerHTML = allSeries.map(s => {
        const thumb = s.thumbnailPath ? `<img class="series-card-thumb" data-thumb="${escapeAttr(s.thumbnailPath)}" src="" alt="">` : `<div class="series-card-thumb-placeholder">&#9733;</div>`;
        const statusClass = `status-${s.status}`;
        const statusLabel = formatStatus(s.status);
        const sc = s.score || 0;
        const score = sc > 0 ? `<span class="score-badge">&#9733; ${sc.toFixed(1)}</span>` : '';
        const tags = (s.tags || []).slice().sort((a, b) => a.localeCompare(b)).slice(0, 3).map(t => `<span class="series-card-tag">${escapeHtml(t)}</span>`).join('');
        const dateInfo = s.startDate ? `<span>${escapeHtml(s.startDate)}</span>` : s.endDate ? `<span>${escapeHtml(s.endDate)}</span>` : '';
        const checked = _selectedIds.has(s.id) ? 'checked' : '';

                    return `<div class="series-card" tabindex="0" onclick="handleCardClick(event, '${s.id}')">
            <input type="checkbox" class="series-card-select" data-id="${s.id}" ${checked} onchange="event.stopPropagation(); updateBatchBar()">
            ${thumb}
            <div class="series-card-info">
                <div class="series-card-title" title="${escapeAttr(s.englishName || 'Untitled')}">${escapeHtml(s.englishName || 'Untitled')}</div>
                <div class="series-card-subtitle" title="${escapeAttr(s.japaneseName || '')}">${escapeHtml(s.japaneseName || '')}</div>
                <div class="series-card-meta">
                    <span class="status-badge ${statusClass}">${statusLabel}</span>
                    ${score} ${dateInfo}
                </div>
                <div class="series-card-tags">${tags}</div>
            </div>
            <button class="favorite-btn ${s.favorite ? 'is-favorite' : ''}" onclick="event.stopPropagation(); toggleFavorite('${s.id}', this)">${s.favorite ? '&#9829;' : '&#9825;'}</button>
        </div>`;
    }).join('');

    const imgs = container.querySelectorAll('img[data-thumb]');
    await Promise.all(Array.from(imgs).map(async img => {
        const thumbPath = img.dataset.thumb;
        if (_thumbnailCache[thumbPath]) {
            img.src = _thumbnailCache[thumbPath];
            return;
        }
        try {
            const dataUrl = await GoAPI.ReadThumbnail(thumbPath);
            if (dataUrl) {
                const keys = Object.keys(_thumbnailCache);
            if (keys.length >= _THUMBNAIL_CACHE_MAX) {
                delete _thumbnailCache[keys[0]];
            }
            _thumbnailCache[thumbPath] = dataUrl;
                img.src = dataUrl;
            }
        } catch(e) {}
    }));
    document.querySelector('.content').scrollTop = _savedListScroll;
    requestAnimationFrame(() => {
        container.style.visibility = '';
        if (_focusedCardId) {
            const card = container.querySelector(`[onclick*="${_focusedCardId}"]`);
            if (card) card.focus();
            _focusedCardId = null;
        }
    });
}

let _savedListScroll = 0;
let _focusedCardId = null;

async function showSeriesDetail(id, fromList) {
    if (fromList) playClickSound();
    try {
        const series = await GoAPI.GetSeries(id);
        if (!series) return;
        currentSeriesId = id;
        window._originalSeries = JSON.parse(JSON.stringify(series));
        window._currentEditData = { englishName: series.englishName || '', japaneseName: series.japaneseName || '', favorite: series.favorite || false, owned: series.owned || false };
        window._currentSynonyms = [...(series.synonyms || [])];
        window._currentLinkedIds = [...(series.linkedIds || [])];
        window._currentTags = [...(series.tags || [])];
        window._currentThumbnail = series.thumbnailPath || '';

        _savedListScroll = document.querySelector('.content').scrollTop;
        document.getElementById('series-list').classList.add('hidden');
        document.getElementById('batch-bar').classList.remove('visible');
        const detail = document.getElementById('series-detail');
        detail.classList.remove('hidden');
        document.querySelector('.content').scrollTop = 0;

        const [linkedResults, all] = await Promise.all([
            series.linkedIds.length > 0
                ? Promise.all(series.linkedIds.map(lid => GoAPI.GetSeries(lid).catch(() => null)))
                : Promise.resolve([]),
            GoAPI.GetAllSeries()
        ]);
        const items = linkedResults.filter(Boolean).map((linked, i) => {
            const lid = series.linkedIds[i];
            const name = escapeHtml(linked.englishName || linked.japaneseName || 'Untitled');
            return `<div class="linked-series-item"><span class="linked-series-name" onclick="showSeriesDetail('${lid}')">${name}</span><span class="remove-link" onclick="event.stopPropagation(); unlinkSeries('${id}', '${lid}')">&times;</span></div>`;
        });
        linkedHtml = items.join('') || '<div style="color: var(--text-muted); font-size: 12px;">No linked series</div>';
        _linkableSeries = all.filter(s => s.id !== id && !(series.linkedIds || []).includes(s.id));

        const thumbHtml = series.thumbnailPath
            ? `<img class="series-detail-thumb" data-thumb="${escapeAttr(series.thumbnailPath)}" src="" alt="">`
            : `<div class="series-detail-thumb-placeholder">&#9733;</div>`;

        detail.innerHTML = `
            <button class="back-btn" onclick="confirmLeave()">&larr; Back to list</button>
            <div class="series-detail-header">
                <div class="thumbnail-upload">
                    ${thumbHtml}
                    <div><button class="btn btn-sm btn-secondary" onclick="uploadThumbnail()">Change Image</button></div>
                </div>
                <div class="series-detail-header-info">
                        <div style="display:flex;align-items:center;gap:2px">
                        <div class="detail-title-editable" id="edit-english-name" onclick="startEditTitle(this, 'englishName')" title="Click to edit">${escapeHtml(series.englishName || 'Untitled')}</div>
                        <button class="owned-btn detail-owned-btn ${series.owned ? 'is-owned' : ''}" onclick="toggleOwned('${id}', this)" title="Owned">${series.owned ? '&#9733;' : '&#9734;'}</button>
                        <button class="favorite-btn detail-favorite-btn ${series.favorite ? 'is-favorite' : ''}" onclick="toggleFavorite('${id}', this)" title="Favorite">${series.favorite ? '&#9829;' : '&#9825;'}</button>
                    </div>
                    <div class="detail-subtitle-editable" id="edit-japanese-name" onclick="startEditTitle(this, 'japaneseName')" title="Click to edit">${escapeHtml(series.japaneseName || '')}</div>
                    <div style="margin-top:12px">
                        <label style="font-size:12px;color:var(--text-secondary)">Status</label>
                        <select id="edit-status" class="status-select status-${series.status}" onchange="updateStatusStyle(this); scheduleAutoSave('${id}', 'status', this.value)">
                            <option value="to-watch" ${series.status==='to-watch'?'selected':''}>To Watch</option>
                            <option value="watched" ${series.status==='watched'?'selected':''}>Watched</option>
                            <option value="in-progress" ${series.status==='in-progress'?'selected':''}>In Progress</option>
                            <option value="abandoned" ${series.status==='abandoned'?'selected':''}>Abandoned</option>
                        </select>
                    </div>
                    <div class="series-detail-header-actions">
                        <button class="btn btn-primary" onclick="saveSeries('${id}')">Save</button>
                        <button class="btn btn-danger" onclick="confirmDelete('${id}')">Delete</button>
                    </div>
                </div>
            </div>
            <div class="form-row">
                <div class="form-group"><label>Start Date</label><input type="text" id="edit-start-date" value="${escapeAttr(series.startDate)}" placeholder="YYYY, YYYY/MM, or YYYY/MM/DD" onblur="scheduleAutoSave('${id}', 'startDate', this.value)" onkeydown="if(event.key==='Enter'){this.blur();}"></div>
                <div class="form-group"><label>End Date</label><input type="text" id="edit-end-date" value="${escapeAttr(series.endDate)}" placeholder="YYYY, YYYY/MM, or YYYY/MM/DD" onblur="scheduleAutoSave('${id}', 'endDate', this.value)" onkeydown="if(event.key==='Enter'){this.blur();}"></div>
                <div class="form-group" style="max-width:120px"><label>Score (0-10)</label><input type="number" id="edit-score" value="${series.score}" min="0" max="10" step="0.1" onblur="scheduleAutoSave('${id}', 'score', parseFloat(this.value) || 0)" onkeydown="if(event.key==='Enter'){this.blur();}"></div>
            </div>
            <div class="form-group">
                <label>Synopsis</label>
                <div class="md-display" id="display-synopsis" onclick="startEditMarkdown('synopsis')" title="Click to edit"></div>
                <textarea id="edit-synopsis" class="md-editor hidden" rows="4" placeholder="Anime synopsis…" spellcheck="false">${escapeHtml(series.synopsis || '')}</textarea>
            </div>
            <div class="form-group">
                <label>Review / Notes</label>
                <div class="md-display" id="display-review" onclick="startEditMarkdown('review')" title="Click to edit"></div>
                <textarea id="edit-review" class="md-editor hidden" rows="3" placeholder="Your personal notes…" spellcheck="false">${escapeHtml(series.review || '')}</textarea>
            </div>
            <div class="form-group">
                <label>Alternative Names</label>
                <div id="synonyms-container">
                    ${(series.synonyms || []).map((syn, i) => `<div class="linked-series-item" style="margin-bottom:4px"><span style="cursor:pointer;padding:2px 4px;border:1px solid transparent;border-radius:3px;box-sizing:border-box;width:100%;display:block;" onclick="editSynonym(${i}, '${id}')" title="Click to edit">${escapeHtml(syn)}</span><span class="remove-link" onclick="removeSynonym(${i})">&times;</span></div>`).join('')}
                </div>
                <div class="add-link-row">
                    <input type="text" id="new-synonym" placeholder="Add alternative name…" onkeydown="if(event.key==='Enter'){event.preventDefault();addSynonym('${id}')}" style="flex:1;padding:6px 8px;background:var(--bg-tertiary);border:1px solid var(--border);border-radius:4px;color:var(--text-primary);font-size:13px;">
                    <button class="btn btn-sm btn-secondary" onclick="addSynonym('${id}')">Add</button>
                </div>
            </div>
            <div class="form-group">
                <label>Tags</label>
                <div class="tags-input" id="tags-input">
                    ${(series.tags || []).map((tag, i) => renderTagChip(tag, i, true)).join('')}
                    <input type="text" id="new-tag-input" placeholder="Add tag…" onkeydown="handleTagKeydown(event, '${id}')">
                </div>
            </div>
            <div class="form-group">
                <label>Linked Series</label>
                <div class="linked-series-list" id="linked-list">${linkedHtml}</div>
                <div class="add-link-row" style="position:relative">
                    <input type="text" id="link-search" placeholder="Search series to link…" onfocus="showLinkSuggestions()" oninput="filterLinkSuggestions()" onkeydown="handleLinkKeydown(event, '${id}')">
                    <div id="link-suggestions" class="link-suggestions hidden"></div>
                    <button class="btn btn-sm btn-secondary" onclick="linkSelectedSeries('${id}')">Link</button>
                </div>
            </div>
        `;

        const createdStr = series.createdAt ? formatTimestamp(series.createdAt) : '';
        const updatedStr = series.updatedAt ? formatTimestamp(series.updatedAt) : '';
        const footer = document.createElement('div');
        footer.className = 'series-detail-footer';
        footer.innerHTML = `<span>Created ${createdStr}</span>${updatedStr ? `<span>Last modified ${updatedStr}</span>` : ''}`;
        detail.appendChild(footer);

        if (series.thumbnailPath) {
            try {
                const dataUrl = await GoAPI.ReadThumbnail(series.thumbnailPath);
                const img = detail.querySelector('.series-detail-thumb');
                if (img && dataUrl) img.src = dataUrl;
            } catch(e) {}
        }
        updateMarkdownDisplay('synopsis');
        updateMarkdownDisplay('review');
        initTextareas();
        document.querySelector('.back-btn')?.focus();
    } catch (e) { console.error('Failed to load series:', e); }
}

const _mdPlaceholders = {
    synopsis: 'Click to add anime synopsis…',
    review: 'Click to add your personal notes…'
};
let _mdEditField = null;

function updateMarkdownDisplay(field) {
    const ta = document.getElementById('edit-' + field);
    const disp = document.getElementById('display-' + field);
    if (!ta || !disp) return;
    const raw = ta.value || '';
    disp.innerHTML = '';
    disp.classList.remove('md-body');
    if (!raw.trim()) {
        disp.classList.add('md-placeholder');
        disp.textContent = _mdPlaceholders[field] || 'Click to add…';
    } else {
        disp.classList.remove('md-placeholder');
        disp.classList.add('md-body');
        disp.appendChild(renderMarkdown(raw));
    }
}

function startEditMarkdown(field) {
    const ta = document.getElementById('edit-' + field);
    const disp = document.getElementById('display-' + field);
    if (!ta || !disp || _mdEditField) return;
    if (!ta.classList.contains('hidden')) return;

    _mdEditField = field;
    ta.dataset.prevValue = ta.value;
    disp.classList.add('hidden');
    ta.classList.remove('hidden');
    autoResizeTextarea(ta);
    if (!ta._hasAutoResize) {
        ta._hasAutoResize = true;
        ta.addEventListener('input', () => autoResizeTextarea(ta));
    }
    ta.focus();

    ta.onblur = () => finishEditMarkdown(field, false);
    ta.onkeydown = (e) => {
        if (e.key === 'Escape') {
            e.preventDefault();
            e.stopPropagation();
            finishEditMarkdown(field, true);
        }
    };
}

function finishEditMarkdown(field, cancel) {
    const ta = document.getElementById('edit-' + field);
    const disp = document.getElementById('display-' + field);
    if (!ta || !disp) return;

    ta.onblur = null;
    ta.onkeydown = null;

    if (cancel) {
        ta.value = ta.dataset.prevValue || '';
    }

    ta.classList.add('hidden');
    disp.classList.remove('hidden');
    if (_mdEditField === field) _mdEditField = null;
    updateMarkdownDisplay(field);
}

let _editingField = null;
let _pendingAutoSave = {};
let _autoSaveTimer = null;

function scheduleAutoSave(id, field, value) {
    _pendingAutoSave[field] = value;
    clearTimeout(_autoSaveTimer);
    _autoSaveTimer = setTimeout(() => flushAutoSave(id), 300);
}

let _autoSaveInProgress = false;

async function flushAutoSave(id) {
    if (_autoSaveInProgress) return;
    _autoSaveInProgress = true;
    try {
        while (Object.keys(_pendingAutoSave).length > 0) {
            const fields = { ..._pendingAutoSave };
            _pendingAutoSave = {};
            const series = await GoAPI.GetSeries(id);
            if (!series) return;
            for (const [k, v] of Object.entries(fields)) {
                series[k] = v;
            }
            await GoAPI.UpdateSeries(id, series);
            window._originalSeries = JSON.parse(JSON.stringify(series));
        }
    } catch (e) {
        console.error('Auto-save failed:', e);
    } finally {
        _autoSaveInProgress = false;
    }
}

function normStr(s) {
    return (s || '').trim().replace(/\r\n/g, '\n').replace(/\s+/g, ' ');
}

function hasUnsavedChanges() {
    if (!window._originalSeries || !currentSeriesId) return false;
    const o = window._originalSeries;
    const e = document.getElementById('edit-english-name');
    const j = document.getElementById('edit-japanese-name');
    if (!e || !j) return false;

    const current = {
        englishName: normStr(window._currentEditData ? window._currentEditData.englishName : e.textContent),
        japaneseName: normStr(window._currentEditData ? window._currentEditData.japaneseName : j.textContent),
        status: document.getElementById('edit-status').value,
        startDate: normStr(document.getElementById('edit-start-date').value),
        endDate: normStr(document.getElementById('edit-end-date').value),
        score: parseFloat(document.getElementById('edit-score').value) || 0,
        synopsis: normStr(document.getElementById('edit-synopsis').value),
        review: normStr(document.getElementById('edit-review').value),
        synonyms: JSON.stringify(window._currentSynonyms || []),
        linkedIds: JSON.stringify(window._currentLinkedIds || []),
        tags: JSON.stringify(window._currentTags || []),
        thumbnailPath: normStr(window._currentThumbnail),
    };

    const saved = {
        englishName: normStr(o.englishName),
        japaneseName: normStr(o.japaneseName),
        status: o.status || '',
        startDate: normStr(o.startDate),
        endDate: normStr(o.endDate),
        score: o.score || 0,
        synopsis: normStr(o.synopsis),
        review: normStr(o.review),
        synonyms: JSON.stringify(o.synonyms || []),
        linkedIds: JSON.stringify(o.linkedIds || []),
        tags: JSON.stringify(o.tags || []),
        thumbnailPath: normStr(o.thumbnailPath),
    };

    return JSON.stringify(current) !== JSON.stringify(saved);
}

let _searchGuardBypassed = false;
let _searchGuardCooldown = false;

function confirmLeave() {
    if (hasUnsavedChanges() && !confirm('You have unsaved changes. Leave without saving?')) return;
    _searchGuardBypassed = true;
    searchSeries();
}

function startEditTitle(el, field) {
    if (_editingField) return;
    _editingField = field;
    const currentValue = window._currentEditData ? window._currentEditData[field] : '';
    const input = document.createElement('input');
    input.type = 'text';
    input.className = el.classList.contains('detail-title-editable') ? 'detail-title-input' : 'detail-subtitle-input';
    input.value = currentValue;
    input.dataset.field = field;
    input.placeholder = field === 'englishName' ? 'English Name' : 'Japanese Name';
    input.onblur = () => finishEditTitle(input, el, field);
    input.onkeydown = (e) => {
        if (e.key === 'Enter') input.blur();
        if (e.key === 'Escape') {
            input.value = currentValue;
            input.blur();
        }
    };
    el.innerHTML = '';
    el.appendChild(input);
    input.focus();
    input.select();
}

function finishEditTitle(input, el, field) {
    const value = input.value.trim();
    if (window._currentEditData) window._currentEditData[field] = value;
    _editingField = null;
    el.innerHTML = escapeHtml(value || (field === 'englishName' ? 'Untitled' : ''));
}

function buildSeriesData(id) {
    return {
        englishName: window._currentEditData ? window._currentEditData.englishName : (document.getElementById('edit-english-name').textContent || ''),
        japaneseName: window._currentEditData ? window._currentEditData.japaneseName : (document.getElementById('edit-japanese-name').textContent || ''),
        status: document.getElementById('edit-status').value,
        startDate: document.getElementById('edit-start-date').value,
        endDate: document.getElementById('edit-end-date').value,
        score: parseFloat(document.getElementById('edit-score').value) || 0,
        synopsis: document.getElementById('edit-synopsis').value,
        review: document.getElementById('edit-review').value,
        synonyms: window._currentSynonyms || [],
        linkedIds: window._currentLinkedIds || [],
        tags: (window._currentTags || []).slice().sort((a, b) => a.localeCompare(b)),
        thumbnailPath: window._currentThumbnail || '',
        favorite: window._currentEditData ? window._currentEditData.favorite : false,
        owned: window._currentEditData ? window._currentEditData.owned : false,
    };
}

async function saveSeries(id) {
    clearTimeout(_autoSaveTimer);
    _pendingAutoSave = {};
    if (!hasUnsavedChanges()) {
        const btn = document.querySelector('.series-detail-header-actions .btn-primary');
        if (btn) {
            const orig = btn.textContent;
            btn.textContent = 'No changes';
            btn.style.opacity = '0.5';
            setTimeout(() => { btn.textContent = orig; btn.style.opacity = ''; }, 1000);
        }
        return;
    }
    const data = buildSeriesData(id);
    try {
        await GoAPI.UpdateSeries(id, data);
        window._currentTags = data.tags;
        window._originalSeries = JSON.parse(JSON.stringify(data));
        await loadTags();
        await searchSeries();
    } catch (e) { console.error('Save failed:', e); alert('Failed to save: ' + e); }
}

let _modalTrigger = null;

function setupModal(overlay) {
    _modalTrigger = document.activeElement;
    overlay.addEventListener('click', (e) => {
        if (e.target === overlay) { overlay.remove(); _modalTrigger?.focus(); }
    });
    overlay.addEventListener('keydown', (e) => {
        if (e.key === 'Escape') {
            overlay.remove();
            _modalTrigger?.focus();
            return;
        }
        if (e.key === 'Tab') {
            const focusable = overlay.querySelectorAll('input, select, textarea, button, [tabindex]:not([tabindex="-1"])');
            if (focusable.length === 0) return;
            const first = focusable[0];
            const last = focusable[focusable.length - 1];
            if (e.shiftKey && document.activeElement === first) {
                e.preventDefault();
                last.focus();
            } else if (!e.shiftKey && document.activeElement === last) {
                e.preventDefault();
                first.focus();
            }
        }
    });
    overlay.tabIndex = -1;
    overlay.focus();
}

function showAddSeries() {
    const overlay = document.createElement('div');
    overlay.className = 'modal-overlay';
    overlay.innerHTML = `<div class="modal">
        <h2>Add New Series</h2>
        <div class="form-group"><label>English Name</label><input type="text" id="new-english-name" placeholder="English title"></div>
        <div class="form-group"><label>Japanese Name</label><input type="text" id="new-japanese-name" placeholder="Japanese title"></div>
        <div class="form-row">
            <div class="form-group"><label>Status</label><select id="new-status"><option value="to-watch">To Watch</option><option value="in-progress">In Progress</option><option value="watched">Watched</option><option value="abandoned">Abandoned</option></select></div>
            <div class="form-row">
                <div class="form-group" style="max-width:120px"><label>Score (0-10)</label><input type="number" id="new-score" min="0" max="10" step="0.1" value="0"></div>
            </div>
        </div>
        <div class="form-row">
            <div class="form-group"><label>Start Date</label><input type="text" id="new-start-date" placeholder="YYYY, YYYY/MM, or YYYY/MM/DD"></div>
            <div class="form-group"><label>End Date</label><input type="text" id="new-end-date" placeholder="YYYY, YYYY/MM, or YYYY/MM/DD"></div>
        </div>
        <div class="form-group"><label>Synopsis</label><textarea id="new-synopsis" rows="3" placeholder="Synopsis…"></textarea></div>
        <div class="modal-actions">
            <button class="btn btn-secondary" onclick="this.closest('.modal-overlay').remove(); _modalTrigger?.focus()">Cancel</button>
            <button class="btn btn-primary" onclick="submitNewSeries()">Create</button>
        </div>
    </div>`;
    document.body.appendChild(overlay);
    setupModal(overlay);
    document.getElementById('new-english-name').focus();
}

async function submitNewSeries() {
    const data = {
        englishName: document.getElementById('new-english-name').value,
        japaneseName: document.getElementById('new-japanese-name').value,
        status: document.getElementById('new-status').value,
        score: parseFloat(document.getElementById('new-score').value) || 0,
        startDate: document.getElementById('new-start-date').value,
        endDate: document.getElementById('new-end-date').value,
        synopsis: document.getElementById('new-synopsis').value,
        synonyms: [],
        linkedIds: [],
        tags: [],
        thumbnailPath: '',
    };
    try {
        await GoAPI.CreateSeries(data);
        document.querySelector('.modal-overlay').remove();
        await loadTags();
        await refreshTotalCount();
        await searchSeries();
    } catch (e) { console.error('Create failed:', e); alert('Failed to create: ' + e); }
}

async function confirmDelete(id) {
    if (!confirm('Are you sure you want to delete this series?')) return;
    try {
        await GoAPI.DeleteSeries(id);
        currentSeriesId = null;
        await loadTags();
        await refreshTotalCount();
        await searchSeries();
    } catch(e) { console.error('Delete failed:', e); }
}

async function uploadThumbnail() {
    try {
        const path = await GoAPI.PickThumbnail();
        if (!path) return;
        window._currentThumbnail = path;
        const dataUrl = await GoAPI.ReadThumbnail(path);
        const img = document.querySelector('.series-detail-thumb, .series-detail-thumb-placeholder');
        if (img && dataUrl) img.outerHTML = `<img class="series-detail-thumb" src="${dataUrl}" alt="">`;
    } catch (e) { console.error('Thumbnail pick failed:', e); }
}

function addSynonym(id) {
    const input = document.getElementById('new-synonym');
    const val = input.value.trim();
    if (!val) return;
    if (!window._currentSynonyms) window._currentSynonyms = [];
    window._currentSynonyms.push(val);
    input.value = '';
    refreshSynonyms(id);
}

function removeSynonym(idx) {
    if (window._currentSynonyms) {
        window._currentSynonyms.splice(idx, 1);
        if (currentSeriesId) refreshSynonyms(currentSeriesId);
    }
}

function refreshSynonyms(id) {
    const container = document.getElementById('synonyms-container');
    if (!container) return;
    container.innerHTML = (window._currentSynonyms || []).map((syn, i) =>
        `<div class="linked-series-item" style="margin-bottom:4px"><span style="cursor:pointer;padding:2px 4px;border:1px solid transparent;border-radius:3px;box-sizing:border-box;width:100%;display:block;" onclick="editSynonym(${i}, '${id}')" title="Click to edit">${escapeHtml(syn)}</span><span class="remove-link" onclick="removeSynonym(${i})">&times;</span></div>`
    ).join('');
}

function editSynonym(idx, id) {
    if (_editingField) return;
    _editingField = 'synonym_' + idx;
    const container = document.getElementById('synonyms-container');
    const items = container.querySelectorAll('.linked-series-item');
    const item = items[idx];
    if (!item) return;
    const span = item.querySelector('span');
    const currentValue = window._currentSynonyms[idx];
    const input = document.createElement('input');
    input.type = 'text';
    input.value = currentValue;
    input.style.cssText = 'padding:2px 4px;background:var(--bg-tertiary);border:1px solid var(--border);border-radius:3px;color:var(--text-primary);font-size:13px;margin:0;box-sizing:border-box;width:100%;';
    span.style.cssText = 'padding:2px 4px;border:1px solid var(--border);border-radius:3px;margin:0;box-sizing:border-box;width:100%;background:var(--bg-tertiary);color:var(--text-primary);';
    const finish = () => {
        const val = input.value.trim();
        if (val && val !== currentValue) {
            window._currentSynonyms[idx] = val;
        }
        _editingField = null;
        refreshSynonyms(id);
    };
    input.onblur = finish;
    input.onkeydown = (e) => {
        if (e.key === 'Enter') input.blur();
        if (e.key === 'Escape') {
            input.value = currentValue;
            input.blur();
        }
    };
    span.replaceWith(input);
    input.focus();
    input.select();
}

function handleTagKeydown(event, id) {
    if (event.key === 'Enter') {
        event.preventDefault();
        const input = document.getElementById('new-tag-input');
        const val = input.value.trim();
        if (!val) return;
        if (!window._currentTags) window._currentTags = [];
        if (!window._currentTags.some(t => t.toLowerCase() === val.toLowerCase())) window._currentTags.push(val);
        input.value = '';
        refreshTagsUI();
        document.getElementById('new-tag-input').focus();
    }
}

function renderTagChip(tag, index, isRemovable) {
    const known = allTags.some(t => t.toLowerCase() === tag.toLowerCase());
    const nameSpan = known
        ? `<span class="tag-search-link" onclick="event.stopPropagation(); searchForTag('${escapeAttr(tag)}')">${escapeHtml(tag)}</span>`
        : `<span class="tag-text">${escapeHtml(tag)}</span>`;
    const remove = isRemovable ? ` <span class="tag-remove" onclick="removeTag(${index})">&times;</span>` : '';
    return `<span class="tag-chip">${nameSpan}${remove}</span>`;
}

function refreshTagsUI() {
    const container = document.getElementById('tags-input');
    if (!container) return;
    container.innerHTML = (window._currentTags || []).map((tag, i) => renderTagChip(tag, i, true)).join('') + `<input type="text" id="new-tag-input" placeholder="Add tag…" onkeydown="handleTagKeydown(event, '${currentSeriesId}')">`;
}

function removeTag(idx) {
    if (window._currentTags) {
        window._currentTags.splice(idx, 1);
        refreshTagsUI();
    }
}

let _linkableSeries = [];
let _selectedLinkIndex = -1;

function renderSuggestionItem(s, i) {
    const display = s.englishName || s.japaneseName || 'Untitled';
    const sub = s.japaneseName && s.englishName ? s.japaneseName : '';
    return `<div class="link-suggestion-item" data-id="${s.id}" data-index="${i}" onclick="selectLinkSuggestion('${s.id}', '${escapeAttr(display)}')">${escapeHtml(display)}${sub ? ' <span style="color:var(--text-muted);font-size:11px">(' + escapeHtml(sub) + ')</span>' : ''}</div>`;
}

function showLinkSuggestions() {
    const input = document.getElementById('link-search');
    const suggestions = document.getElementById('link-suggestions');
    const query = input.value.toLowerCase().trim();

    if (query) {
        filterLinkSuggestions();
        return;
    }

    const recent = _linkableSeries.slice().sort((a, b) => {
        return new Date(b.updatedAt) - new Date(a.updatedAt);
    }).slice(0, 5);

    if (recent.length === 0) {
        suggestions.classList.add('hidden');
        return;
    }

    _selectedLinkIndex = -1;
    suggestions.innerHTML = recent.map((s, i) => renderSuggestionItem(s, i)).join('');
    suggestions.classList.remove('hidden');
}

function filterLinkSuggestions() {
    const input = document.getElementById('link-search');
    const suggestions = document.getElementById('link-suggestions');
    const query = input.value.toLowerCase().trim();

    if (!query) {
        showLinkSuggestions();
        return;
    }

    const filtered = _linkableSeries.filter(s => {
        const name = (s.englishName || '').toLowerCase();
        const jpName = (s.japaneseName || '').toLowerCase();
        const syns = (s.synonyms || []).map(x => x.toLowerCase());
        return name.includes(query) || jpName.includes(query) || syns.some(x => x.includes(query));
    });

    if (filtered.length === 0) {
        suggestions.classList.add('hidden');
        return;
    }

    _selectedLinkIndex = -1;
    suggestions.innerHTML = filtered.map((s, i) => renderSuggestionItem(s, i)).join('');
    suggestions.classList.remove('hidden');
}

function selectLinkSuggestion(id, name) {
    document.getElementById('link-search').value = name;
    window._selectedLinkId = id;
    document.getElementById('link-suggestions').classList.add('hidden');
}

function handleLinkKeydown(event, currentId) {
    const suggestions = document.getElementById('link-suggestions');
    const items = suggestions.querySelectorAll('.link-suggestion-item');

    if (event.key === 'ArrowDown') {
        event.preventDefault();
        _selectedLinkIndex = Math.min(_selectedLinkIndex + 1, items.length - 1);
        updateLinkSelection(items);
    } else if (event.key === 'ArrowUp') {
        event.preventDefault();
        _selectedLinkIndex = Math.max(_selectedLinkIndex - 1, 0);
        updateLinkSelection(items);
    } else if (event.key === 'Enter') {
        event.preventDefault();
        if (_selectedLinkIndex >= 0 && items[_selectedLinkIndex]) {
            const id = items[_selectedLinkIndex].dataset.id;
            const name = items[_selectedLinkIndex].textContent;
            selectLinkSuggestion(id, name);
        }
        linkSelectedSeries(currentId);
    } else if (event.key === 'Escape') {
        suggestions.classList.add('hidden');
    }
}

function updateLinkSelection(items) {
    items.forEach((item, i) => {
        item.classList.toggle('selected', i === _selectedLinkIndex);
    });
}

async function linkSelectedSeries(id) {
    const linkId = window._selectedLinkId;
    if (!linkId) return;
    try {
        const data = buildSeriesData(id);
        const linkedIds = [...(data.linkedIds || [])];
        if (!linkedIds.includes(linkId)) linkedIds.push(linkId);
        data.linkedIds = linkedIds;
        await GoAPI.UpdateSeries(id, data);
        window._originalSeries = JSON.parse(JSON.stringify(data));
        window._selectedLinkId = null;
        document.getElementById('link-search').value = '';
        document.getElementById('link-suggestions').classList.add('hidden');
        await loadTags();
        await showSeriesDetail(id);
    } catch(e) { console.error('Link failed:', e); }
}

async function unlinkSeries(id, linkId) {
    try {
        const data = buildSeriesData(id);
        data.linkedIds = (data.linkedIds || []).filter(lid => lid !== linkId);
        await GoAPI.UpdateSeries(id, data);
        window._originalSeries = JSON.parse(JSON.stringify(data));
        await showSeriesDetail(id);
    } catch(e) { console.error('Unlink failed:', e); }
}

async function exportCSV() {
    try {
        const data = await GoAPI.ExportCSV();
        const blob = new Blob([data], { type: 'text/csv' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'naraberu-export.csv';
        a.click();
        URL.revokeObjectURL(url);
    } catch (e) { console.error('Export failed:', e); alert('Export failed: ' + e); }
}

async function importCSV(event) {
    const file = event.target.files[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = async function(e) {
        try {
            const text = e.target.result;
            const result = await GoAPI.ImportCSV(text);
            let msg = 'Imported ' + result.count + ' series.';
            if (result.skipped && result.skipped.length > 0) {
                msg += '\nSkipped (already exist): ' + result.skipped.join(', ');
            }
            alert(msg);
            await loadTags();
            await refreshTotalCount();
            await searchSeries();
        } catch (err) { console.error('Import failed:', err); alert('Import failed: ' + err); }
    };
    reader.readAsText(file);
    event.target.value = '';
}

function updateStatusStyle(select) {
    select.className = 'status-select status-' + select.value;
}

function formatStatus(status) {
    return { 'to-watch': 'To Watch', 'watched': 'Watched', 'in-progress': 'In Progress', 'abandoned': 'Abandoned' }[status] || status;
}

function formatTimestamp(isoStr) {
    const d = new Date(isoStr);
    if (isNaN(d.getTime())) return '';
    const pad = n => String(n).padStart(2, '0');
    return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

async function toggleFavorite(id, btn) {
    try {
        const series = await GoAPI.ToggleFavorite(id);
        if (!series) return;
        btn.classList.toggle('is-favorite', series.favorite);
        btn.innerHTML = series.favorite ? '&#9829;' : '&#9825;';
        if (window._currentEditData) window._currentEditData.favorite = series.favorite;
    } catch(e) { console.error('Toggle favorite failed:', e); }
}

async function toggleOwned(id, btn) {
    try {
        const series = await GoAPI.ToggleOwned(id);
        if (!series) return;
        btn.classList.toggle('is-owned', series.owned);
        btn.innerHTML = series.owned ? '&#9733;' : '&#9734;';
        if (window._currentEditData) window._currentEditData.owned = series.owned;
    } catch(e) { console.error('Toggle owned failed:', e); }
}

let _currentAudioFile = 'pick.mp3';
let _currentMuteAudio = false;

function playClickSound() {
    if (_currentMuteAudio) return;
    const audio = document.getElementById('click-sound');
    if (!audio || !audio.src) return;
    audio.volume = 0.15;
    audio.currentTime = 0;
    audio.play().catch(() => {});
}

function searchForTag(tag) {
    document.getElementById('search-input').value = '';
    document.getElementById('filter-status').value = '';
    document.getElementById('filter-favorites').checked = false;
    document.getElementById('filter-linked').checked = false;
    document.getElementById('filter-owned').checked = false;
    currentFilter.query = '';
    currentFilter.tags = [tag];
    currentFilter.favoritesOnly = false;
    currentFilter.linkedOnly = false;
    currentFilter.ownedOnly = false;
    renderTagFilters();
    searchSeries();
}

function setThemeClass(theme) {
    const validThemes = ['dark', 'light', 'pink-dark', 'pink-light', 'red-dark', 'red-light', 'green-dark', 'green-light', 'blue-dark', 'blue-light'];
    document.documentElement.className = validThemes.includes(theme) ? `theme-${theme}` : 'theme-dark';
}

async function applyTheme() {
    try {
        const theme = await GoAPI.GetTheme();
        setThemeClass(theme);
    } catch(e) {
        setThemeClass('dark');
    }
}

async function applyAudioSrc(file) {
    const audio = document.getElementById('click-sound');
    if (!audio) return;
    if (!file || file === 'none') {
        audio.src = '';
        return;
    }
    try {
        const src = await GoAPI.ReadAudioFile(file);
        audio.src = src || '';
    } catch (e) {
        audio.src = '';
    }
}

async function loadAudioSettings() {
    try {
        _currentAudioFile = await GoAPI.GetAudioFile();
        _currentMuteAudio = await GoAPI.GetMuteAudio();
        await applyAudioSrc(_currentAudioFile);
    } catch(e) {
        _currentAudioFile = 'pick.mp3';
        _currentMuteAudio = false;
        await applyAudioSrc(_currentAudioFile);
    }
}

async function setTheme(theme) {
    try {
        await GoAPI.SetTheme(theme);
        setThemeClass(theme);
    } catch(e) { console.error('Failed to set theme:', e); }
}

async function showSettings() {
    let currentTheme = 'dark';
    let currentAudioFile = 'pick.mp3';
    let currentMuteAudio = false;
    let audioFiles = ['pick.mp3'];
    try {
        currentTheme = await GoAPI.GetTheme();
        currentAudioFile = await GoAPI.GetAudioFile();
        currentMuteAudio = await GoAPI.GetMuteAudio();
        audioFiles = await GoAPI.ListAudioFiles();
    } catch(e) {}

    const overlay = document.createElement('div');
    overlay.className = 'modal-overlay';
    overlay.innerHTML = `<div class="modal" style="width:520px">
        <h2>Settings</h2>

        <div class="settings-section">
            <h3 style="font-size:13px;color:var(--text-secondary);margin-bottom:12px;text-transform:uppercase;letter-spacing:0.5px;">Appearance</h3>
            <div class="form-group">
                <label>Color Theme</label>
                <select id="settings-theme" class="status-select" onchange="settingsThemeChanged(this.value)">
                    <option value="dark" ${currentTheme==='dark'?'selected':''}>Dark</option>
                    <option value="light" ${currentTheme==='light'?'selected':''}>Light</option>
                    <option value="pink-dark" ${currentTheme==='pink-dark'?'selected':''}>Pink (Dark)</option>
                    <option value="pink-light" ${currentTheme==='pink-light'?'selected':''}>Pink (Light)</option>
                    <option value="red-dark" ${currentTheme==='red-dark'?'selected':''}>Red (Dark)</option>
                    <option value="red-light" ${currentTheme==='red-light'?'selected':''}>Red (Light)</option>
                    <option value="green-dark" ${currentTheme==='green-dark'?'selected':''}>Green (Dark)</option>
                    <option value="green-light" ${currentTheme==='green-light'?'selected':''}>Green (Light)</option>
                    <option value="blue-dark" ${currentTheme==='blue-dark'?'selected':''}>Blue (Dark)</option>
                    <option value="blue-light" ${currentTheme==='blue-light'?'selected':''}>Blue (Light)</option>
                </select>
            </div>
        </div>

        <div class="settings-section" style="margin-top:20px;">
            <h3 style="font-size:13px;color:var(--text-secondary);margin-bottom:12px;text-transform:uppercase;letter-spacing:0.5px;">Audio</h3>
            <div class="form-group">
                <label>Click Sound</label>
                <select id="settings-audio-file" onchange="settingsAudioFileChanged(this.value)">
                    ${audioFiles.map(f => `<option value="${escapeAttr(f)}" ${currentAudioFile===f?'selected':''}>${escapeHtml(f)}</option>`).join('')}
                    <option value="none" ${currentAudioFile==='none'?'selected':''}>None</option>
                </select>
            </div>
            <div class="form-group">
                <label style="display:inline-flex;align-items:center;gap:8px;cursor:pointer;">
                    <input type="checkbox" id="settings-mute" ${currentMuteAudio?'checked':''} onchange="settingsMuteChanged(this.checked)">
                    Mute click sound
                </label>
            </div>
        </div>

        <div class="modal-actions" style="margin-top:20px;">
            <button class="btn btn-secondary" onclick="this.closest('.modal-overlay').remove(); _modalTrigger?.focus()">Close</button>
        </div>
    </div>`;
    document.body.appendChild(overlay);
    setupModal(overlay);
}

async function settingsThemeChanged(theme) {
    await GoAPI.SetTheme(theme);
    setThemeClass(theme);
}

async function settingsAudioFileChanged(file) {
    await GoAPI.SetAudioFile(file);
    _currentAudioFile = file;
    await applyAudioSrc(file);
}

async function settingsMuteChanged(mute) {
    await GoAPI.SetMuteAudio(mute);
    _currentMuteAudio = mute;
}

function handleCardClick(event, id) {
    if (event.target.classList.contains('series-card-select')) return;
    _focusedCardId = id;
    showSeriesDetail(id, true);
}

function updateBatchBar() {
    _selectedIds.clear();
    document.querySelectorAll('.series-card-select:checked').forEach(cb => _selectedIds.add(cb.dataset.id));
    const bar = document.getElementById('batch-bar');
    const count = document.getElementById('batch-count');
    if (_selectedIds.size > 0) {
        bar.classList.add('visible');
        count.textContent = `${_selectedIds.size} selected`;
    } else {
        bar.classList.remove('visible');
    }
}

function toggleSelectAll(checked) {
    document.querySelectorAll('.series-card-select').forEach(cb => {
        cb.checked = checked;
        if (checked) _selectedIds.add(cb.dataset.id);
        else _selectedIds.delete(cb.dataset.id);
    });
    updateBatchBar();
}

function clearSelection() {
    _selectedIds.clear();
    document.querySelectorAll('.series-card-select').forEach(cb => cb.checked = false);
    document.getElementById('select-all').checked = false;
    updateBatchBar();
}

async function batchDelete() {
    const ids = Array.from(_selectedIds);
    if (ids.length === 0) return;
    if (!confirm(`Delete ${ids.length} series? This cannot be undone.`)) return;
    try {
        for (const id of ids) {
            await GoAPI.DeleteSeries(id);
        }
        clearSelection();
        await loadTags();
        await refreshTotalCount();
        await searchSeries();
    } catch(e) { console.error('Batch delete failed:', e); alert('Failed to delete: ' + e); }
}

async function batchAddTag() {
    const input = document.getElementById('batch-tag-input');
    const tag = input.value.trim();
    if (!tag) return;
    const ids = Array.from(_selectedIds);
    if (ids.length === 0) return;
    try {
        for (const id of ids) {
            const series = await GoAPI.GetSeries(id);
            if (!series) continue;
            const tags = [...(series.tags || [])];
            if (!tags.some(t => t.toLowerCase() === tag.toLowerCase())) tags.push(tag);
            series.tags = tags.slice().sort((a, b) => a.localeCompare(b));
            await GoAPI.UpdateSeries(id, series);
        }
        input.value = '';
        await loadTags();
        await searchSeries();
    } catch(e) { console.error('Batch tag add failed:', e); alert('Failed to add tag: ' + e); }
}

function autoResizeTextarea(el) {
    el.style.height = 'auto';
    el.style.height = el.scrollHeight + 'px';
}

function initTextareas() {
    document.querySelectorAll('.form-group textarea').forEach(ta => {
        if (ta.offsetParent === null) return;
        autoResizeTextarea(ta);
        if (!ta._hasAutoResize) {
            ta._hasAutoResize = true;
            ta.addEventListener('input', () => autoResizeTextarea(ta));
        }
    });
}

function escapeHtml(str) {
    if (!str) return '';
    return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;').replace(/`/g, '&#96;');
}

function escapeAttr(str) {
    if (!str) return '';
    return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/'/g, '&#39;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

document.addEventListener('click', (e) => {
    const suggestions = document.getElementById('link-suggestions');
    const searchInput = document.getElementById('link-search');
    if (suggestions && !suggestions.contains(e.target) && e.target !== searchInput) {
        suggestions.classList.add('hidden');
    }
});

document.addEventListener('keydown', (e) => {
    if (document.querySelector('.modal-overlay')) return;
    const tag = document.activeElement?.tagName;
    const isInput = tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT';

    if (e.key === '/' && !isInput) {
        e.preventDefault();
        document.getElementById('search-input').focus();
    }
    if (e.key === 'n' && !isInput) {
        e.preventDefault();
        showAddSeries();
    }
    if (e.key === 'Escape') {
        if (_mdEditField) {
            e.preventDefault();
            e.stopPropagation();
            finishEditMarkdown(_mdEditField, true);
            return;
        }
        if (!document.getElementById('series-detail').classList.contains('hidden')) {
            confirmLeave();
        }
    }
    if (e.key === 'Backspace' && !isInput) {
        e.preventDefault();
        document.getElementById('search-input').value = '';
        document.getElementById('filter-status').value = '';
        document.getElementById('filter-favorites').checked = false;
        document.getElementById('filter-linked').checked = false;
        document.getElementById('filter-owned').checked = false;
        currentFilter.query = '';
        currentFilter.tags = [];
        currentFilter.favoritesOnly = false;
        currentFilter.linkedOnly = false;
        currentFilter.ownedOnly = false;
        renderTagFilters();
        searchSeries();
    }
    if (isInput) return;
    const detail = document.getElementById('series-detail');
    const list = document.getElementById('series-list');
    if (!list.classList.contains('hidden') && allSeries.length > 0) {
        if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
            e.preventDefault();
            const cards = list.querySelectorAll('.series-card');
            let idx = Array.from(cards).indexOf(document.activeElement);
            if (idx === -1) idx = e.key === 'ArrowDown' ? 0 : cards.length - 1;
            else idx = e.key === 'ArrowDown' ? Math.min(idx + 1, cards.length - 1) : Math.max(idx - 1, 0);
            cards[idx]?.focus();
            cards[idx]?.scrollIntoView({ block: 'nearest' });
        }
        if (e.key === 'Enter' && document.activeElement?.classList.contains('series-card')) {
            e.preventDefault();
            document.activeElement.click();
        }
    }
});

(async function init() {
    try {
        await waitForService();
    } catch (e) {
        console.error(e);
        document.getElementById('result-count').textContent = 'Failed to connect to backend';
        return;
    }
    document.getElementById('sort-desc').checked = true;
    await applyTheme();
    await loadAudioSettings();
    await loadTags();
    await refreshTotalCount();
    await searchSeries();
})();
