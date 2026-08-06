"use strict";

import { appSettings, LangHelper, UT, PT, PT_REFRESH, SearchType } from './settings.js';
import { TextProcessor } from './pali-script.js';
import { CustomDict } from './custom-dict.js';
import { Util } from './util.js';

class PaliAnalysis {
    constructor() {
        this.openWindows = new Map();
        this.settings = { hoverDelay: 500 };
        this.hoverTimer = null;
        this._selPopup = null;
    }
    init(vManager, dictClient, loadLookupCb) {
        this.vManager = vManager;
        this.dictClient = dictClient;
        this.loadLookupCb = loadLookupCb;
        this._initSelectionPopup();
    }

    // ─── Text-selection popup (Feature 3) ──────────────────────────────────
    _initSelectionPopup() {
        // Create the floating popup button
        this._selPopup = $('<button id="sel-search-popup">🔍 ค้นหา</button>');
        $('body').append(this._selPopup);

        // Show popup on mouseup if there's a selection in the text pane
        $(document).on('mouseup', e => {
            // small delay so selection is settled
            setTimeout(() => this._checkSelection(e), 80);
        });

        // Hide popup on mousedown (user starts a new selection)
        $(document).on('mousedown', e => {
            if (!$(e.target).closest('#sel-search-popup').length) {
                this._selPopup.hide();
            }
        });

        // On popup click: load selected text into search
        this._selPopup.on('click', () => {
            const sel = window.getSelection();
            const text = sel ? sel.toString().trim() : '';
            if (text && this._searchCb) {
                this._searchCb(text);
            }
            this._selPopup.hide();
        });
    }

    // Register callback for when popup is clicked
    setSearchCallback(cb) { this._searchCb = cb; }

    _checkSelection(e) {
        // Don't show inside the analysis window or search bars
        if ($(e.target).closest('.analysis-window, #dict-area, .search-bar, input, textarea').length) return;

        const sel = window.getSelection();
        if (!sel || sel.rangeCount === 0) return;
        const text = sel.toString().trim();
        if (text.length < 2) { this._selPopup.hide(); return; }

        // Position popup near end of selection
        const range = sel.getRangeAt(0);
        const rect = range.getBoundingClientRect();
        const popW = 110, popH = 30;
        let left = rect.right + 8;
        let top = rect.top + window.scrollY - 4;
        if (left + popW > window.innerWidth - 10) left = rect.left - popW - 8;
        if (top < 4) top = rect.bottom + window.scrollY + 4;

        this._selPopup.css({ left, top }).show();
    }

    // ─── Analysis window ────────────────────────────────────────────────────
    async showWindow(wordEvent) {
        const target = $(wordEvent.currentTarget);
        const pane = this.vManager.curPane;
        const newWindow = await this.createAnalysisWindow(target.text(), pane);
        this.closeWindow(pane);
        this.openWindow(pane, newWindow, target);
    }

    /** Open analysis window directly with a word (no click event needed).
     *  Used by the selection popup — stays in the current reading pane. */
    async showWindowForWord(word) {
        const pane = this.vManager.curPane;
        if (pane == null) return; // no reading pane open yet
        const newWindow = await this.createAnalysisWindow(word, pane);
        this.closeWindow(pane);
        // openWindow without a target element (no highlight/scroll)
        $(this.vManager.getPaneRoot(pane)).append(newWindow);
        this.openWindows.set(pane, [newWindow, null]);
        this.registerClicks(newWindow);
    }


    async createAnalysisWindow(word, pane) {
        const rawWord = word.trim();

        // Editable word input (Feature 2)
        const wordInput = $('<input type="text" class="word-input">')
            .val(TextProcessor.convert(TextProcessor.convertFromMixed(rawWord), appSettings.get('paliScript')));

        const lookupIcons = Object.entries(appSettings.searchTypeProp).map(([type, prop]) => {
            return $(`<i class="${prop.iconClass} lookup-icon" word="${rawWord}" type="${type}"></i>`);
        });
        
        // Custom Dict Button
        const customDictIcon = $('<i class="fas fa-plus-circle custom-dict-icon" title="พจนานุกรมเพิ่มเอง" style="cursor:pointer; margin-right: 10px; font-size: 1.1em; opacity: 0.8; color: #28a745;"></i>');
        
        const closeIcon = $('<i class="far fa-times-circle close-icon"></i>').attr('pane', pane);
        const headerRow = $('<div/>').addClass('header').append(lookupIcons, wordInput, customDictIcon, closeIcon);

        // Drag resize handle at top (Feature 2)
        const resizeHandle = $('<div class="resize-handle"><div class="resize-grip"></div></div>');

        // Dict results in its own scrollable container (Feature 2)
        const entries = await this.dictClient.searchWordInline(rawWord);
        const breakup = $('<div/>').addClass('breakups').html(entries.breakups);
        const dictScroll = $('<div class="dict-scroll"/>');
        const dictElem = $('<div/>').addClass('dict-inline').html(entries.matches);
        dictScroll.append(dictElem);

        const win = $('<div/>').addClass('analysis-window').addClass(appSettings.get('analysisStyle'))
            .append(resizeHandle, headerRow, breakup, dictScroll);

        // Wire up editable word: re-search on Enter or after pause
        let reSearchTimer = null;
        wordInput.on('input', () => {
            clearTimeout(reSearchTimer);
            reSearchTimer = setTimeout(async () => {
                const newWord = wordInput.val().trim();
                if (newWord.length < 2) return;
                const newEntries = await this.dictClient.searchWordInline(newWord);
                breakup.html(newEntries.breakups);
                dictElem.html(newEntries.matches);
                // update lookup icons
                win.find('.lookup-icon').each((_, el) => $(el).attr('word', newWord));
            }, 500);
        }).on('keydown', e => {
            if (e.key === 'Enter') wordInput.trigger('input'); // immediate
        });

        // Wire up resize handle (drag top edge to resize)
        this._initResizeDrag(win, resizeHandle);

        return win;
    }

    _initResizeDrag(win, handle) {
        let startY = 0, startH = 0;
        handle.on('mousedown', e => {
            e.preventDefault();
            startY = e.clientY;
            startH = win.outerHeight();
            $(document).on('mousemove.resize', ev => {
                const delta = startY - ev.clientY; // drag up = bigger
                const newH = Math.max(80, Math.min(window.innerHeight * 0.8, startH + delta));
                win.css({ height: newH + 'px', minHeight: 'unset', maxHeight: 'unset' });
            }).on('mouseup.resize', () => {
                $(document).off('mousemove.resize mouseup.resize');
            });
        });
    }

    openWindow(pane, newWindow, target) {
        $(this.vManager.getPaneRoot(pane)).append(newWindow);
        this.openWindows.set(pane, [newWindow, target]);
        target.addClass('highlighted').get(0).scrollIntoView({ block: 'center' });
        this.registerClicks(newWindow);
    }
    closeWindow(pane) {
        if (this.openWindows.has(pane)) {
            const [window, target] = this.openWindows.get(pane);
            window.remove();
            if (target) target.removeClass('highlighted'); // target may be null (opened from selection popup)
            this.openWindows.delete(pane);
        }
    }

    registerClicks(window) {
        window.on('click', '.lookup-icon', e => {
            const icon = $(e.currentTarget);
            this.loadLookupCb(icon.attr('word'), icon.attr('type'));
        }).on('click', '.close-icon', e => {
            this.closeWindow($(e.currentTarget).attr('pane'));
        }).on('click', '.custom-dict-icon', e => {
            const word = window.find('.word-input').val() || window.find('.lookup-icon').first().attr('word');
            this.showCustomDictDialog(word);
        });
    }

    showCustomDictDialog(initialWord = '') {
        const dialogContent = $('<div/>').css({ display: 'flex', flexDirection: 'column', height: '400px' });
        dialogContent.append(`<h3>พจนานุกรมเพิ่มเอง (Custom Dictionary)</h3>`);
        
        const formDiv = $('<div/>').css({ display: 'flex', gap: '10px', marginBottom: '15px', paddingBottom: '15px', borderBottom: '1px solid #ccc' });
        const wordInput = $('<input type="text" placeholder="คำศัพท์">').css({ flex: 1 }).val(initialWord);
        const meaningInput = $('<input type="text" placeholder="ความหมาย">').css({ flex: 2 });
        const addBtn = $('<button>เพิ่ม</button>');
        formDiv.append(wordInput, meaningInput, addBtn);
        
        const filterInput = $('<input type="text" placeholder="🔍 ค้นหาคำศัพท์ที่บันทึกไว้...">').css({ width: '100%', padding: '5px', marginBottom: '10px', boxSizing: 'border-box' });
        
        const listDiv = $('<div/>').css({ flex: 1, overflowY: 'auto' });
        
        const renderList = () => {
            listDiv.empty();
            let entries = CustomDict.getAll();
            
            const filterText = filterInput.val().trim().toLowerCase();
            if (filterText) {
                entries = entries.filter(e => e.word.toLowerCase().includes(filterText) || e.meaning.toLowerCase().includes(filterText));
            }
            
            if (entries.length === 0) {
                listDiv.append('<p style="color: gray;">ยังไม่มีคำศัพท์ที่เพิ่มเอง</p>');
                return;
            }
            entries.forEach(entry => {
                const itemDiv = $('<div/>').css({ display: 'flex', justifyContent: 'space-between', padding: '5px', borderBottom: '1px solid #eee' });
                const textSpan = $('<span/>').css({ flex: 1 }).text(`${entry.word} : ${entry.meaning}`);
                const btnSpan = $('<span/>').css({ display: 'flex', gap: '10px' });
                
                const editIcon = $('<i class="far fa-edit" title="แก้ไข"></i>').css({ cursor: 'pointer', color: '#007bff' });
                const delIcon = $('<i class="far fa-trash-alt" title="ลบ"></i>').css({ cursor: 'pointer', color: '#dc3545' });
                
                editIcon.on('click', () => {
                    const newWord = prompt('แก้ไขคำศัพท์:', entry.word);
                    if (newWord === null) return;
                    const newMeaning = prompt('แก้ไขความหมาย:', entry.meaning);
                    if (newMeaning === null) return;
                    if (confirm('คุณต้องการบันทึกการแก้ไขนี้ใช่หรือไม่?')) {
                        CustomDict.edit(entry.id, newWord, newMeaning);
                        renderList();
                    }
                });
                
                delIcon.on('click', () => {
                    if (confirm(`คุณต้องการลบคำว่า "${entry.word}" ใช่หรือไม่?`)) {
                        CustomDict.delete(entry.id);
                        renderList();
                    }
                });
                
                btnSpan.append(editIcon, delIcon);
                itemDiv.append(textSpan, btnSpan);
                listDiv.append(itemDiv);
            });
        };
        
        addBtn.on('click', () => {
            const w = wordInput.val().trim();
            const m = meaningInput.val().trim();
            if (w && m) {
                CustomDict.add(w, m);
                wordInput.val('');
                meaningInput.val('');
                renderList();
            } else {
                alert('กรุณากรอกทั้งคำศัพท์และความหมาย');
            }
        });
        
        renderList();
        filterInput.on('input', renderList);
        
        dialogContent.append(formDiv, filterInput, listDiv);
        Util.showDialog('generic-dialog', dialogContent);
    }
}

export const paliAnalysis = new PaliAnalysis();