export function Home() {
  return (
    <div className="p-4 flex flex-col gap-4 justify-center items-center h-full opacity-40">
      <span className="text-9xl">🌷</span>
      <h1 className="text-5xl text-gray-600">Welcome to Tulip</h1>
      <h1 className="text-2xl text-gray-500">Shortcut reference:</h1>
      <table className="border-collapse border border-slate-500 table-auto">
        <thead>
          <tr>
            <th className="border border-slate-600 px-4">Key</th>
            <th className="border border-slate-600 px-4">Action</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td className="border border-slate-700 px-4">j/k</td>
            <td className="border border-slate-700 px-4">Up/Down in FlowList</td>
          </tr>
          <tr>
            <td className="border border-slate-700 px-4">s</td>
            <td className="border border-slate-700 px-4">Focus search bar</td>
          </tr>
          <tr>
            <td className="border border-slate-700 px-4">esc</td>
            <td className="border border-slate-700 px-4">Unfocus search bar</td>
          </tr>
          <tr>
            <td className="border border-slate-700 px-4">i/o</td>
            <td className="border border-slate-700 px-4">Toggle flag in/out filters</td>
          </tr>
        </tbody>
      </table>
      {/* <h1 className="text-3xl font-bold pt-2 pb-4"></h1> */}
    </div>
  );
}
