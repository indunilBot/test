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

export default function PebbleDBExplorer() {
  const [dbs, setDbs] = useState<string[]>([]);
  const [selectedDb, setSelectedDb] = useState<string>('');
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [viewMode, setViewMode] = useState<'tree' | 'list'>('tree');
  const [expandedKeys, setExpandedKeys] = useState<Set<string>>(new Set());
  const [keysByDb, setKeysByDb] = useState<{ [db: string]: { [prefix: string]: string[] } }>({});
  const [values, setValues] = useState<{ [cacheKey: string]: string }>({});
  const [isAddingConnection, setIsAddingConnection] = useState(false);
  const [connectionName, setConnectionName] = useState('');
  const [connectionPath, setConnectionPath] = useState('');
  const [connectionError, setConnectionError] = useState<string | null>(null);

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
      throw new Error('Wails backend is not available. Run the desktop app to add connections.');
    }
    await backend.AddConnection(name, path);
  };

  const resetConnectionForm = () => {
    setConnectionName('');
    setConnectionPath('');
    setConnectionError(null);
  };

  const openConnectionForm = () => {
    resetConnectionForm();
    setIsAddingConnection(true);
  };

  const closeConnectionForm = () => {
    setIsAddingConnection(false);
    resetConnectionForm();
  };

  const handleBrowseForPath = async () => {
    const runtime = (window as any).runtime;
    if (runtime?.OpenDirectoryDialog) {
      const selected = await runtime.OpenDirectoryDialog({ title: 'Select PebbleDB Directory' });
      if (selected) {
        setConnectionPath(selected);
        setConnectionError(null);
      }
    } else {
      setConnectionError('Directory picker is not available in this environment. Please paste the path manually.');
    }
  };

  const submitConnection = async () => {
    const trimmedName = connectionName.trim();
    const trimmedPath = connectionPath.trim();
    if (!trimmedName) {
      setConnectionError('Enter a connection name.');
      return;
    }
    if (!trimmedPath) {
      setConnectionError('Provide the PebbleDB directory path.');
      return;
    }

    try {
      await safeAddConnection(trimmedName, trimmedPath);
      const newDbs: string[] = await safeGetDatabases();
      setDbs(newDbs);
      setSelectedDb(trimmedName);
      closeConnectionForm();
    } catch (err: any) {
      setConnectionError(err?.message ?? String(err));
    }
  };

  useEffect(() => {
    safeGetDatabases().then((res: string[]) => setDbs(res ?? []));
  }, []);

  useEffect(() => {
    if (selectedDb && !keysByDb[selectedDb]) {
      safeGetKeysByPrefix(selectedDb).then((prefixes: { [prefix: string]: string[] } | null) => {
        if (prefixes) {
          setKeysByDb(prev => ({ ...prev, [selectedDb]: prefixes }));
        }
      });
    }
  }, [selectedDb]);

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

  const getKeysByPrefix = () => keysByDb[selectedDb] || {};

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
    <div className="relative flex h-screen bg-slate-100 text-slate-900">
      <div className="w-64 bg-[#19334D] text-slate-100 flex flex-col">
        <div className="p-4 border-b border-slate-700/60">
          <div className="flex items-center gap-2 mb-4">
            <Database className="w-6 h-6 text-slate-200" />
            <h1 className="text-lg font-semibold tracking-wide">GPaw Explorer</h1>
          </div>
          <button
            onClick={openConnectionForm}
            className="w-full inline-flex items-center justify-center gap-2 rounded-lg bg-[#003C67] px-4 py-2 font-medium text-white transition hover:opacity-90"
          >
            <Plus className="w-4 h-4" />
            New Connection
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-2">
          <div className="px-2 py-1 text-xs font-semibold uppercase tracking-[0.2em] text-slate-400">Databases</div>
          {dbs.map(db => (
            <div
              key={db}
              onClick={() => setSelectedDb(db)}
              className={`mb-1 flex cursor-pointer items-center gap-2 rounded-lg px-3 py-2 text-sm transition ${
                selectedDb === db
                  ? 'bg-[#003C67] text-white shadow-inner'
                  : 'hover:bg-slate-800/40'
              }`}
            >
              <Database className="w-4 h-4" />
              <span className="flex-1 truncate">{db}</span>
              <span className="text-xs text-slate-300">
                {Object.keys(keysByDb[db] || {}).reduce((sum, p) => sum + (keysByDb[db][p]?.length || 0), 0)}
              </span>
            </div>
          ))}
          {dbs.length === 0 && (
            <div className="px-3 py-2 text-sm text-slate-300">
              No databases connected. Add a new connection.
            </div>
          )}
        </div>

        <div className="border-t border-slate-700/60 p-4">
          <button className="w-full inline-flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm transition hover:bg-slate-800/50">
            <Settings className="w-4 h-4" />
            Settings
          </button>
        </div>
      </div>

      <div className="flex flex-1 flex-col bg-white">
        <div className="flex items-center gap-3 border-b border-slate-200 bg-white p-4">
          <div className="relative flex-1">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-slate-400" />
            <input
              type="text"
              placeholder="Search keys..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full rounded-lg border border-slate-300 bg-white py-2 pl-10 pr-4 text-sm shadow-sm transition focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-500"
            />
          </div>
          <button className="inline-flex items-center gap-2 rounded-lg border border-slate-300 bg-sky-100 px-4 py-2 text-sm font-medium text-sky-800 transition hover:bg-sky-200">
            <Filter className="w-4 h-4" />
            Filter
          </button>
          <button
            onClick={handleRefresh}
            className="inline-flex items-center gap-2 rounded-lg border border-slate-300 bg-sky-100 px-4 py-2 text-sm font-medium text-sky-800 transition hover:bg-sky-200"
          >
            <RefreshCw className="w-4 h-4" />
            Refresh
          </button>
          <button className="inline-flex items-center gap-2 rounded-lg bg-[#003C67] px-4 py-2 text-sm font-medium text-white transition hover:opacity-90">
            <Plus className="w-4 h-4" />
            Add Key
          </button>
        </div>

        <div className="flex flex-1 min-h-0">
          <div className="flex w-80 flex-col border-r border-slate-200 bg-white">
            <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 text-sm font-medium text-slate-600">
              <span>
                Keys ({viewMode === 'tree'
                  ? Object.keys(getKeysByPrefix()).reduce((sum, p) => sum + getKeysByPrefix()[p].length, 0)
                  : filteredKeys().length})
              </span>
              <div className="flex gap-2">
                <button
                  onClick={() => setViewMode('tree')}
                  className={`rounded-md px-3 py-1 text-xs font-semibold transition ${
                    viewMode === 'tree'
                      ? 'bg-sky-100 text-sky-800'
                      : 'text-slate-500 hover:bg-slate-100'
                  }`}
                >
                  Tree
                </button>
                <button
                  onClick={() => setViewMode('list')}
                  className={`rounded-md px-3 py-1 text-xs font-semibold transition ${
                    viewMode === 'list'
                      ? 'bg-sky-100 text-sky-800'
                      : 'text-slate-500 hover:bg-slate-100'
                  }`}
                >
                  List
                </button>
              </div>
            </div>

            <div className="flex-1 overflow-y-auto p-3 space-y-2">
              {viewMode === 'tree'
                ? Object.entries(getKeysByPrefix()).map(([prefix, keys]) => (
                    <div key={prefix} className="space-y-1">
                      <div
                        onClick={() => toggleExpand(prefix)}
                        className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1 text-sm font-medium text-slate-600 transition hover:bg-sky-100"
                      >
                        {expandedKeys.has(prefix) ? (
                          <ChevronDown className="h-4 w-4 text-slate-500" />
                        ) : (
                          <ChevronRight className="h-4 w-4 text-slate-500" />
                        )}
                        <Key className="h-4 w-4 text-sky-700" />
                        <span>{prefix || 'default'}</span>
                        <span className="ml-auto text-xs text-slate-400">{keys.length}</span>
                      </div>
                      {expandedKeys.has(prefix) && (
                        <div className="ml-6 space-y-1">
                          {keys.map(key => (
                            <div
                              key={key}
                              onClick={() => setSelectedKey(key)}
                              className={`flex cursor-pointer items-center gap-2 rounded-md px-2 py-1 text-sm transition ${
                                selectedKey === key
                                  ? 'bg-sky-100 text-sky-800'
                                  : 'text-slate-600 hover:bg-slate-100'
                              }`}
                            >
                              <FileText className="h-4 w-4" />
                              <span className="truncate">{key}</span>
                            </div>
                          ))}
                        </div>
                      )}
                    </div>
                  ))
                : filteredKeys().map(key => (
                    <div
                      key={key}
                      onClick={() => setSelectedKey(key)}
                      className={`flex cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-sm transition ${
                        selectedKey === key
                          ? 'bg-sky-100 text-sky-800'
                          : 'text-slate-600 hover:bg-slate-100'
                      }`}
                    >
                      <Key className="h-4 w-4" />
                      <span className="truncate">{key}</span>
                    </div>
                  ))}
            </div>
          </div>

          <div className="flex-1 overflow-y-auto bg-white">
            {selectedKey ? (
              <div className="p-6 space-y-6">
                <div className="flex items-center justify-between border-b border-slate-200 pb-4">
                  <h2 className="text-xl font-semibold text-slate-900">{selectedKey}</h2>
                  <div className="flex gap-2">
                    <button className="inline-flex items-center gap-2 rounded-md border border-slate-300 bg-white px-3 py-1 text-sm transition hover:bg-slate-100">
                      <Download className="h-4 w-4" />
                      Export
                    </button>
                    <button className="inline-flex items-center gap-2 rounded-md border border-rose-400 bg-rose-50 px-3 py-1 text-sm text-rose-600 transition hover:bg-rose-100">
                      <Trash2 className="h-4 w-4" />
                      Delete
                    </button>
                  </div>
                </div>
                <div className="flex flex-wrap items-center gap-6 text-sm text-slate-600">
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
                          Type: <span className="font-mono text-sky-700">{type}</span>
                        </span>
                        <span>
                          Size: <span className="font-mono">{size}</span>
                        </span>
                      </>
                    );
                  })()}
                </div>
                <div className="rounded-xl bg-[#19334D] p-4 shadow-inner">
                  <pre className="overflow-auto text-sm font-mono text-emerald-300">
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
                <div className="flex gap-3">
                  <button className="inline-flex items-center gap-2 rounded-lg bg-[#003C67] px-4 py-2 text-sm font-medium text-white transition hover:opacity-90">
                    Save Changes
                  </button>
                  <button className="inline-flex items-center gap-2 rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-600 transition hover:bg-slate-100">
                    Cancel
                  </button>
                </div>
              </div>
            ) : (
              <div className="flex h-full items-center justify-center text-slate-400">
                <div className="text-center space-y-2">
                  <Database className="mx-auto h-16 w-16 opacity-50" />
                  <p className="text-lg font-medium">Select a key to view its value</p>
                  <p className="text-sm">Choose from the list on the left</p>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>

      {isAddingConnection && (
        <div className="absolute inset-0 z-40 flex items-center justify-center bg-slate-900/70 px-4">
          <div className="w-full max-w-xl rounded-2xl bg-white p-6 shadow-2xl">
            <div className="mb-4">
              <h2 className="text-xl font-semibold text-slate-900">Add PebbleDB Connection</h2>
              <p className="mt-1 text-sm text-slate-500">
                Paste the database directory path or browse to it. Example:
                <code className="ml-1 rounded bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
                  /Users/user/Desktop/indunil/paw/Go/paw-corenet-layer/coredb/pebbledb/slot-db
                </code>
              </p>
            </div>

            <div className="space-y-4">
              <div>
                <label className="text-sm font-medium text-slate-700">Connection Name</label>
                <input
                  value={connectionName}
                  onChange={(e) => setConnectionName(e.target.value)}
                  className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-500"
                  placeholder="e.g. slot-db"
                />
              </div>

              <div>
                <label className="text-sm font-medium text-slate-700">Database Path</label>
                <div className="mt-1 flex gap-2">
                  <input
                    value={connectionPath}
                    onChange={(e) => setConnectionPath(e.target.value)}
                    className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm shadow-sm focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-500"
                    placeholder="/path/to/pebbledb"
                  />
                  <button
                    onClick={handleBrowseForPath}
                    className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm font-medium text-slate-600 transition hover:bg-slate-100"
                  >
                    Browse
                  </button>
                </div>
                {connectionPath && (
                  <p className="mt-2 text-xs text-slate-500">
                    Selected path: <span className="font-mono text-slate-600">{connectionPath}</span>
                  </p>
                )}
              </div>

              {connectionError && (
                <div className="rounded-lg bg-rose-50 px-3 py-2 text-sm text-rose-600">
                  {connectionError}
                </div>
              )}

              <div className="flex justify-end gap-3">
                <button
                  onClick={closeConnectionForm}
                  className="rounded-lg border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-600 transition hover:bg-slate-100"
                >
                  Cancel
                </button>
                <button
                  onClick={submitConnection}
                  className="rounded-lg bg-[#003C67] px-4 py-2 text-sm font-semibold text-white transition hover:opacity-90"
                >
                  Connect
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
