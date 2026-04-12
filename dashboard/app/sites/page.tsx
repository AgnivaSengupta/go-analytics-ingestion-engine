import DashboardLayout from "@/components/DashboardLayout";
import { Button } from "@/components/ui/button";


function SiteCard() {
  return (
    <div className="w-50 h-37.5 border border-primary rounded-md">
      PaperTrails
    </div>
  )
}

export default function Sites() {
  return (
    <DashboardLayout>
      <main className="min-w-6xl">
        <div className="flex justify-between items-center mb-10">
          <div>
            <h1 className="font-serif text-4xl tracking-wide">Sites</h1>
          </div>
          
          <Button size='lg' className="text-lg font-serif tracking-widest px-4 py-4 cursor-pointer"> + Add Site</Button>
        </div>
        
        <div>
          <SiteCard/>
        </div>
        
      </main>
    </DashboardLayout>
  )
}