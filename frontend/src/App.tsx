import React, { useState, useEffect } from 'react';
import {
  Database,
  Search,
  FileText,
  Trash2,
  Plus,
  RefreshCw,
  ChevronRight,
  ChevronDown,
  Settings,
  Key,
  Filter,
  Download,
} from 'lucide-react';
import * as backend from '../wailsjs/go/main/App';
import './App.css';

export default function PebbleDBExplorer() {
  const [dbs, setDbs] = useState<string[]>([]);
  const [selectedDb, setSelectedDb] = useState<string>('');
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [viewMode, setViewMode] = useState<'tree' | 'list'>('tree');
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(new Set());
  const [keysByDb, setKeysByDb] = useState<{ [db: string]: { [prefix: string]: string[] } }>({});
  const [values, setValues] = useState<{ [cacheKey: string]: string }>({});

  const hasWailsBackend = () => Boolean((window as any).go?.main?.App);

  const safeGetDatabases = async () => {
    if (hasWailsBackend()) {
      const result = await backend.GetDatabases();
      return Array.isArray(result) ? result : [];
    }
    console.warn('Wails backend not available. Returning empty database list.');
    return [] as string[];
  };

  const safeGetKeysByPrefix = async (db: string) => {
    if (hasWailsBackend()) {
      return backend.GetKeysByPrefix(db);
    }
    console.warn('Wails backend not available. Skipping key fetch.');
    return null;
  };

  const safeGetValue = async (db: string, key: string) => {
    if (hasWailsBackend()) {
      return backend.GetValue(db, key);
    }
    console.warn('Wails backend not available. Returning empty value.');
    return '';
  };

  const safeAddConnection = async (name: string, path: string) => {
    if (!hasWailsBackend()) {
      console.warn('Wails backend not available. Connection will not be persisted.');
      return;
    }
    await backend.AddConnection(name, path);
  };

  // Fetch databases on load
  useEffect(() => {
    safeGetDatabases().then((res: string[]) => setDbs(res ?? []));
  }, []);

  // Fetch keys when DB selected
  useEffect(() => {
    if (selectedDb && !keysByDb[selectedDb]) {
      safeGetKeysByPrefix(selectedDb).then((prefixes: { [prefix: string]: string[] } | null) => {
        if (prefixes) {
          setKeysByDb(prev => ({ ...prev, [selectedDb]: prefixes }));
        }
      });
    }
  }, [selectedDb]);

  // Fetch value when key selected
  useEffect(() => {
    if (selectedDb && selectedKey) {
      const cacheKey = `${selectedDb}_${selectedKey}`;
      if (!values[cacheKey]) {
        safeGetValue(selectedDb, selectedKey).then((val: string) => {
          if (val) {
            setValues(prev => ({ ...prev, [cacheKey]: val }));
          }
        });
      }
    }
  }, [selectedDb, selectedKey]);

  // Handle new connection
  const handleNewConnection = async () => {
    const name = prompt('Enter connection name:');
    if (!name) return;
    const runtime = (window as any).runtime;
    const path = runtime?.OpenDirectoryDialog
      ? await runtime.OpenDirectoryDialog({ title: 'Select PebbleDB Directory' })
      : prompt('Enter the PebbleDB directory path:');
    if (path) {
      try {
        await safeAddConnection(name, path);
        const newDbs: string[] = await safeGetDatabases();
        setDbs(newDbs);
        setSelectedDb(name);
      } catch (err) {
        alert(`Error adding connection: ${err}`);
      }
    }
  };

  // Get prefixes for selected DB
  const getKeysByPrefix = () => keysByDb[selectedDb] || {};

  // Filtered keys for list view
  const filteredKeys = () => {
    const prefixes = getKeysByPrefix();
    let allKeys: string[] = [];
    Object.values(prefixes).forEach(keys => {
      allKeys = allKeys.concat(keys);
    });
    return allKeys.filter(key => key.toLowerCase().includes(searchTerm.toLowerCase()));
  };

  const toggleExpand = (prefix: string) => {
    const newExpanded = new Set(expandedKeys);
    if (newExpanded.has(prefix)) {
      newExpanded.delete(prefix);
    } else {
      newExpanded.add(prefix);
    }
    setExpandedKeys(newExpanded);
  };

  // Refresh keys for current DB
  const handleRefresh = () => {
    if (selectedDb) {
      safeGetKeysByPrefix(selectedDb).then((prefixes: { [prefix: string]: string[] } | null) => {
        if (prefixes) {
          setKeysByDb(prev => ({ ...prev, [selectedDb]: prefixes }));
        }
      });
    }
  };

  return (
    <div className="app-layout">
      <div className="sidebar">
        <div className="sidebar-header">
          <div className="sidebar-title">
            <Database className="icon-lg" style={{ color: '#ADD8E6' }} />
            <h1>GPaw Explorer</h1>
          </div>
          <button onClick={handleNewConnection} className="sidebar-button">
            <Plus className="icon-sm" />
            New Connection
          </button>
        </div>

        <div className="sidebar-content">
          <div className="sidebar-section-title">DATABASES</div>
          {dbs.map(db => (
            <div
              key={db}
              onClick={() => setSelectedDb(db)}
              className={`database-item ${selectedDb === db ? 'active' : ''}`}
            >
              <Database className="icon-sm" />
              <span className="database-name">{db}</span>
              <span className="database-count">
                {Object.keys(keysByDb[db] || {}).reduce((sum, p) => sum + (keysByDb[db][p]?.length || 0), 0)}
              </span>
            </div>
          ))}
          {dbs.length === 0 && (
            <div className="empty-state">No databases connected. Add a new connection.</div>
          )}
        </div>

        <div className="sidebar-footer">
          <button className="sidebar-footer-button">
            <Settings className="icon-sm" />
            Settings
          </button>
        </div>
      </div>

      <div className="main-panel">
        <div className="toolbar">
          <div className="search-wrapper">
            <Search className="icon-md search-icon" />
            <input
              type="text"
              placeholder="Search keys..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="search-input"
            />
          </div>
          <button className="toolbar-button-secondary">
            <Filter className="icon-sm" />
            Filter
          </button>
          <button onClick={handleRefresh} className="toolbar-button-secondary">
            <RefreshCw className="icon-sm" />
            Refresh
          </button>
          <button className="toolbar-button-primary">
            <Plus className="icon-sm" />
            Add Key
          </button>
        </div>

        <div className="content-split">
          <div className="keys-panel">
            <div className="keys-panel-header">
              <span>
                Keys ({viewMode === 'tree' ? Object.keys(getKeysByPrefix()).reduce((sum, p) => sum + getKeysByPrefix()[p].length, 0) : filteredKeys().length})
              </span>
              <div className="view-toggle-group">
                <button
                  onClick={() => setViewMode('tree')}
                  className={`view-toggle-button ${viewMode === 'tree' ? 'active' : ''}`}
                >
                  Tree
                </button>
                <button
                  onClick={() => setViewMode('list')}
                  className={`view-toggle-button ${viewMode === 'list' ? 'active' : ''}`}
                >
                  List
                </button>
              </div>
            </div>

            <div className="keys-scroll">
              {viewMode === 'tree' ? (
                Object.entries(getKeysByPrefix()).map(([prefix, keys]) => (
                  <div key={prefix} className="prefix-group">
                    <div onClick={() => toggleExpand(prefix)} className="prefix-header">
                      {expandedKeys.has(prefix) ? (
                        <ChevronDown className="icon-sm" />
                      ) : (
                        <ChevronRight className="icon-sm" />
                      )}
                      <Key className="icon-sm" style={{ color: '#003C67' }} />
                      <span className="prefix-label">{prefix || 'default'}</span>
                      <span className="prefix-count">{keys.length}</span>
                    </div>
                    {expandedKeys.has(prefix) && (
                      <div className="prefix-keys">
                        {keys.map(key => (
                          <div
                            key={key}
                            onClick={() => setSelectedKey(key)}
                            className={`key-item ${selectedKey === key ? 'active' : ''}`}
                          >
                            <FileText className="icon-sm" />
                            {key}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                ))
              ) : (
                filteredKeys().map(key => (
                  <div
                    key={key}
                    onClick={() => setSelectedKey(key)}
                    className={`keys-list-item ${selectedKey === key ? 'active' : ''}`}
                  >
                    <Key className="icon-sm" />
                    <span>{key}</span>
                  </div>
                ))
              )}
            </div>
          </div>

          <div className="value-panel">
            {selectedKey ? (
              <div className="value-wrapper">
                <div className="value-header">
                  <h2 className="value-title">{selectedKey}</h2>
                  <div className="value-actions">
                    <button>
                      <Download className="icon-sm" />
                      Export
                    </button>
                    <button className="danger">
                      <Trash2 className="icon-sm" />
                      Delete
                    </button>
                  </div>
                </div>
                <div className="value-meta">
                  {(() => {
                    const cacheKey = `${selectedDb}_${selectedKey}`;
                    const rawValue = values[cacheKey] || '';
                    let type = 'Unknown';
                    let size = '0 bytes';
                    if (rawValue) {
                      size = `${rawValue.length} bytes`;
                      try {
                        JSON.parse(rawValue);
                        type = 'Object';
                      } catch {
                        type = 'String';
                      }
                    }
                    return (
                      <>
                        <span>
                          Type: <span className="font-mono" style={{ color: '#003C67' }}>{type}</span>
                        </span>
                        <span>
                          Size: <span className="font-mono">{size}</span>
                        </span>
                      </>
                    );
                  })()}
                </div>

                <div className="value-code">
                  <pre>
                    {(() => {
                      const cacheKey = `${selectedDb}_${selectedKey}`;
                      const rawValue = values[cacheKey];
                      if (!rawValue) return 'Loading...';
                      try {
                        const parsed = JSON.parse(rawValue);
                        return JSON.stringify(parsed, null, 2);
                      } catch {
                        return rawValue;
                      }
                    })()}
                  </pre>
                </div>

                <div className="value-footer">
                  <button className="primary-action">Save Changes</button>
                  <button className="secondary-action">Cancel</button>
                </div>
              </div>
            ) : (
              <div className="value-empty-state">
                <div className="value-empty-state-content">
                  <Database className="icon-lg" style={{ opacity: 0.5 }} />
                  <p>Select a key to view its value</p>
                  <p style={{ fontSize: '0.9rem', marginTop: '0.35rem' }}>Choose from the list on the left</p>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
