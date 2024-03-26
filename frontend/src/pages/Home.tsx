const shortcutTableData = [
  [
    { key: 'j/k', action: 'Down/Up in FlowList' },
    { key: 'h/l', action: 'Up/Down in Flow' },
    { key: 's', action: 'Focus (s)earch bar' },
    { key: 'esc', action: 'Unfocus search bar' },
  ],
  [
    { key: 'a', action: 'L(a)st 5 ticks' },
    { key: 'c', action: '(C)lear time selection' },
    { key: 'r', action: '(R)efresh flows' },
  ],
  [
    { key: 'd', action: '(D)iff view' },
    { key: 'f', action: 'Load flow to (f)irst diff slot' },
    { key: 'e', action: 'Load flow to s(e)cond diff slot' },
    { key: 'g', action: '(G)raph view' },
  ],
  [
    { key: 'w', action: 'Scroll to current flo(w) in flow list' },
    { key: 'i/o', action: 'Toggle flag in/out filters' },
    { key: 't', action: 'Toggle s(t)arred filters' },
    { key: 'x', action: 'Star selected flow' },
  ]
];

const generateShortcutTable = (data: { key: string; action: string; }[][]) => {
  return (
        <div className="flex flex-row gap-4">
          {data.map((table, tableIndex) => (
            <table key={tableIndex} className="border-collapse border border-slate-500 table-auto">
              <thead>
                <tr>
                  <th className="border border-slate-600 px-4">Key</th>
                  <th className="border border-slate-600 px-4">Action</th>
                </tr>
              </thead>
              <tbody>
              {table.map((row, rowIndex) => (
                <tr key={rowIndex}>
                  {Object.entries(row).map((cell, cellIndex) => (
                    <td className="border border-slate-700 px-4" key={cellIndex}>
                      {cell[1]}
                    </td>
                  ))}
                </tr>
              ))}
              </tbody>
            </table>
          ))}
        </div>
  );
};


export function Home() {
  return (
    <div className="p-4 flex flex-col gap-4 justify-center items-center h-full opacity-40">
      <span className="text-9xl">🌷</span>
      <h1 className="text-5xl text-gray-600">Welcome to Tulip</h1>
      <h1 className="text-2xl text-gray-500">Shortcut reference:</h1>
      {generateShortcutTable(shortcutTableData)}
      {/* <h1 className="text-3xl font-bold pt-2 pb-4"></h1> */}
    </div>
  );
}
