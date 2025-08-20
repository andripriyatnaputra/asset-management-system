// File: src/pages/AddAssetPage.tsx
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import apiClient from '../src/services/api';

export default function AddAssetPage() {
  const navigate = useNavigate();
  const [assetTag, setAssetTag] = useState('');
  const [name, setName] = useState('');
  const [assetType, setAssetType] = useState('Laptop');
  const [status, setStatus] = useState('In Stock');
  const [purchaseDate, setPurchaseDate] = useState('');
  const [initialPrice, setInitialPrice] = useState(0);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    try {
      await apiClient.post('/assets', {
        name,
        asset_tag: assetTag,
        asset_type: assetType,
        status,
        purchase_date: new Date(purchaseDate).toISOString(),
        initial_price: Number(initialPrice),
      });
      // Jika berhasil, kembali ke halaman daftar aset
      navigate('/assets');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Gagal menambahkan aset.');
      console.error(err);
    }
  };

  return (
    <div>
      <h1 className="text-2xl font-bold mb-4 text-gray-800">Tambah Aset Baru</h1>
      <div className="bg-white p-6 rounded-lg shadow-md">
        <form onSubmit={handleSubmit}>
          {error && <p className="text-red-500 bg-red-100 p-3 rounded mb-4">{error}</p>}
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* Kolom Kiri */}
            <div>
              <label className="block text-gray-700 mb-2">Nama Aset</label>
              <input type="text" value={name} onChange={e => setName(e.target.value)} className="w-full p-2 border rounded" required />
            </div>
            <div>
              <label className="block text-gray-700 mb-2">Tag Aset</label>
              <input type="text" value={assetTag} onChange={e => setAssetTag(e.target.value)} className="w-full p-2 border rounded" required />
            </div>
            <div>
              <label className="block text-gray-700 mb-2">Tipe Aset</label>
              <select value={assetType} onChange={e => setAssetType(e.target.value)} className="w-full p-2 border rounded">
                <option>Laptop</option>
                <option>Monitor</option>
                <option>Server</option>
                <option>Printer</option>
                <option>Lainnya</option>
              </select>
            </div>
            
            {/* Kolom Kanan */}
            <div>
              <label className="block text-gray-700 mb-2">Status</label>
              <select value={status} onChange={e => setStatus(e.target.value)} className="w-full p-2 border rounded">
                <option>In Stock</option>
                <option>In Repair</option>
                <option>Broken</option>
              </select>
            </div>
            <div>
              <label className="block text-gray-700 mb-2">Tanggal Pembelian</label>
              <input type="date" value={purchaseDate} onChange={e => setPurchaseDate(e.target.value)} className="w-full p-2 border rounded" required />
            </div>
            <div>
              <label className="block text-gray-700 mb-2">Harga Awal (IDR)</label>
              <input type="number" value={initialPrice} onChange={e => setInitialPrice(Number(e.target.value))} className="w-full p-2 border rounded" required />
            </div>
          </div>

          <div className="mt-6 flex justify-end">
            <button type="button" onClick={() => navigate('/assets')} className="bg-gray-300 hover:bg-gray-400 text-gray-800 font-bold py-2 px-4 rounded mr-2">
              Batal
            </button>
            <button type="submit" className="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded">
              Simpan Aset
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}