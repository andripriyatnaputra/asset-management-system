// File: src/pages/AssetTypesPage.tsx
import { useManagementLogic } from '../hooks/useManagementLogic';

import { Button } from "../src/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../src/components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "../src/components/ui/dialog";
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "../src/components/ui/alert-dialog";
import { Input } from "../src/components/ui/input";
import { Label } from "../src/components/ui/label";

export default function AssetTypesPage() {
  const title = "Tipe Aset";
  const apiEndpoint = "/asset-types";
  
  // Memanggil "mesin" logika terpusat kita
  const {
    data, isLoading,
    isDialogOpen, setIsDialogOpen,
    isConfirmOpen, setIsConfirmOpen,
    currentItem, itemName, setItemName, itemToDelete,
    handleOpenDialog, handleCloseDialog, handleSave,
    handleDeleteClick, handleConfirmDelete
  } = useManagementLogic(title, apiEndpoint);

  return (
    <div className="container mx-auto py-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-3xl font-bold">Manajemen {title}</h1>
        <Button onClick={() => handleOpenDialog(null)}>+ Tambah Baru</Button>
      </div>

      <div className="bg-white p-4 border rounded-lg">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>Nama</TableHead>
              <TableHead className="text-right">Aksi</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow><TableCell colSpan={3} className="text-center">Loading...</TableCell></TableRow>
            ) : (
              data.map((item) => (
                <TableRow key={item.id}>
                  <TableCell>{item.id}</TableCell>
                  <TableCell className="font-medium">{item.name}</TableCell>
                  <TableCell className="text-right space-x-2">
                    <Button variant="outline" size="sm" onClick={() => handleOpenDialog(item)}>Edit</Button>
                    <Button variant="destructive" size="sm" onClick={() => handleDeleteClick(item)}>Delete</Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* Dialog untuk Tambah/Edit */}
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{currentItem ? 'Edit' : 'Tambah'} {title}</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <Label htmlFor="name">Nama</Label>
            <Input id="name" value={itemName} onChange={(e) => setItemName(e.target.value)} />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={handleCloseDialog}>Batal</Button>
            <Button onClick={handleSave}>Simpan</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Dialog Konfirmasi Delete */}
      <AlertDialog open={isConfirmOpen} onOpenChange={setIsConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Apakah Anda Yakin?</AlertDialogTitle>
            <AlertDialogDescription>
              Tindakan ini akan menghapus {title.toLowerCase()} "{itemToDelete?.name}".
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Batal</AlertDialogCancel>
            <AlertDialogAction onClick={handleConfirmDelete}>Ya, Hapus</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}