import { Button } from "./ui/button";

export default function Header() {
  return (
    <header className="w-full sticky top-0  flex items-center justify-between p-4 px-10 border-b border-border mb-5">
      <div className="font-serif tracking-wider text-3xl">
        Flux
      </div>
      
      <div className="flex gap-4 items-center">
        
        <Button size='lg' className="text-sm p-4 cursor-pointer">Sign In</Button>
      </div>
    </header>
  )
}