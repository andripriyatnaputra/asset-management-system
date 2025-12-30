import { useEffect, useState } from "react";
import { getAuditLogs } from "@/services/complianceService";

export default function AuditSummaryTable() {
  const [logs, setLogs] = useState<any[]>([]);

  useEffect(() => {
    getAuditLogs(5).then(setLogs).catch(console.error);
  }, []);

  return (
    <div className="p-4 shadow rounded bg-white col-span-3">
      <h3 className="text-lg font-semibold mb-2">Recent Audit Logs</h3>
      <table className="w-full text-sm">
        <thead>
          <tr className="text-left border-b">
            <th>Entity</th>
            <th>Action</th>
            <th>User</th>
            <th>Time</th>
          </tr>
        </thead>
        <tbody>
          {logs.map((l, i) => (
            <tr key={i} className="border-b">
              <td>{l.entity_name}</td>
              <td>{l.action}</td>
              <td>{l.actor_name ?? "—"}</td>
              <td>{new Date(l.created_at).toLocaleString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
