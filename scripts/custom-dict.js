/**
 * Custom Dictionary Manager
 * Stores user-defined dictionary entries in localStorage.
 */
export class CustomDict {
    static STORAGE_KEY = 'customDictionary';

    // Get all custom entries
    static getAll() {
        try {
            const data = localStorage.getItem(this.STORAGE_KEY);
            if (data) {
                return JSON.parse(data);
            }
        } catch (e) {
            console.error('Failed to load custom dictionary', e);
        }
        return [];
    }

    // Save all custom entries
    static saveAll(entries) {
        try {
            localStorage.setItem(this.STORAGE_KEY, JSON.stringify(entries));
        } catch (e) {
            console.error('Failed to save custom dictionary', e);
        }
    }

    // Add a new entry
    static add(word, meaning) {
        const entries = this.getAll();
        const id = Date.now().toString(); // unique ID
        entries.push({ id, word, meaning });
        this.saveAll(entries);
        return id;
    }

    // Edit an existing entry
    static edit(id, word, meaning) {
        const entries = this.getAll();
        const index = entries.findIndex(e => e.id === id);
        if (index > -1) {
            entries[index] = { id, word, meaning };
            this.saveAll(entries);
            return true;
        }
        return false;
    }

    // Delete an entry
    static delete(id) {
        let entries = this.getAll();
        entries = entries.filter(e => e.id !== id);
        this.saveAll(entries);
    }

    // Search for a word
    static search(queryWord) {
        const entries = this.getAll();
        const matches = [];
        const lowerQuery = queryWord.toLowerCase();
        
        // Exact and prefix search
        for (const entry of entries) {
            const lowerWord = entry.word.toLowerCase();
            if (lowerWord.startsWith(lowerQuery) || lowerQuery.startsWith(lowerWord) || lowerWord.includes(lowerQuery)) {
                matches.push({
                    dictName: 'custom',
                    word: entry.word,
                    meaning: entry.meaning,
                    rowid: entry.id, // For UI purposes
                    distance: lowerWord === lowerQuery ? 0 : 1 // Prioritize exact match
                });
            }
        }
        return matches;
    }
}
